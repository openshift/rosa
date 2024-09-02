#!/bin/bash

# override_rosacli_build will override rosacli build for the coming testing with dicated $ROSACLI_BUILD
override_rosacli_build () {
  # make a temp dir for rosa download
  rosaDownloadTempDir=$(mktemp -d)
  cd $rosaDownloadTempDir

  # get the rosa downboad binary according to the version
  wget https://github.com/openshift/rosa/releases/download/$ROSACLI_BUILD/rosa_Linux_x86_64.tar.gz


  tar -xvf rosa_Linux_x86_64.tar.gz
  chmod +x ./rosa

  # override the PATH 
  echo "[CI] Overriding the rosa PATH in the image with new PATH $rosaDownloadTempDir/rosa"
  export PATH=$rosaDownloadTempDir:$PATH

  # verify the rosa override
  # Didn't use rosa version to verify because there is a known issue in old versions which will cause nil pointer error for the command in some env
  echo "[CI] Verify current rosa is using the overrided PATH $rosaDownloadTempDir/rosa"
  current_rosa_path=$(which rosa)
  if [[ "$current_rosa_path" != "$rosaDownloadTempDir/rosa" ]];then
    echo "[CI] rosa override failed. Current rosa is using $current_rosa_path not from $rosaDownloadTempDir/rosa"
    exit 1
  fi
    echo "[CI] rosa is overrided with build $ROSACLI_BUILD"
  
  # go back to previous dir
  cd -
}

# configure_aws will setup aws credentials for the CI job run
# It requires two parameters 
# the first one for aws credential file path which is required
# the second one for aws region which is required
# the third one for second aws cred info for shared vpc scenario which is optional
# usage: configure_aws <aws file path> <region> 
configure_aws () {
  # configure aws region
  if [[ -z "$1" ]] || [[ -z "$2" ]]; then
    echo "ERROR: aws credential file path and region is required. Please call command like $ configure_aws <credential path> <region>"
    exit 1
  fi
  echo "[CI] Configured AWS region to ${REGION}"
  # configure aws credetials
  awscred=$1
  if [[ -f "${awscred}" ]]; then
    export AWS_SHARED_CREDENTIALS_FILE="${awscred}"
    export AWS_DEFAULT_REGION="$2"
  else
    echo "ERROR: aws credential file $1 doesn't exist"
    exit 1
  fi
}

# configure_aws_shared_vpc will configure the aws account for shared vpc scenario
# usage: configure_aws_shared_vpc <shared vpc aws credential file>
configure_aws_shared_vpc () {
  if [[ -z "$1" ]]; then
    echo "ERROR: please provide the shared vpc credential file when call configure_aws_shared_vpc"
    exit 1
  fi
  sharedVPCCredFile=$1
  echo "[CI] Got awscred_shared_account set $sharedVPCCredFile"

  if [[ -f $sharedVPCCredFile ]];then
    export SHARED_VPC_AWS_SHARED_CREDENTIALS_FILE=$sharedVPCCredFile
  else
    echo "ERROR: the shared vpc credential file $sharedVPCCredFile doesn't exist. Please check"
    exit 1
  fi
}

# rosa_login configure rosa login
# first parameter should be env which is required
# second parameter should be token which is required
# usage: rosa_login <env> <token>
rosa_login () {
  if [[ -z "$1" ]] || [[ -z "$2" ]]; then
    echo "ERROR: both env and token are required for rosa login"
    exit 1
  fi
  ocmENV=$1
  ocmToken=$2
  echo "[CI] Running rosa/ocm login based on env $1"
  rosa login --env "${ocmENV}" --token "${ocmToken}"
  ocm login --url "${ocmENV}" --token "${ocmToken}"
}

# generate_label_filter_switch is used to generate label-filter for the test run
# IMPORTANCE is step env for job running
# LABEL_FILTER is global env setting but it can be override by parameter
# if there is parameter when call this function, it will override LABEL_FILTER
# Need to clarify the LABEL_FILTER_SWITCH in steps script 
generate_label_filter_switch () {
  label_filter=${LABEL_FILTER}
  if [ ! -z "$1" ]; then
    echo "[CI] Got parameter label filter: $1 . Override the global setting now."
    label_filter=$1
  fi
  if [[ ! -z "$IMPORTANCE" ]]; then
    label_filter="$label_filter&&${IMPORTANCE}" 
  fi
  LABEL_FILTER_SWITCH="--ginkgo.label-filter '${label_filter}'"
}

# generate_junit is used to generate junit file path
# RUN_TIME needs to be set in the step or job level
# SHARED_DIR is global env for the CI run, no need to set
# TEST_PROFILE is optional in job level
# JUNIT_XML need to be clarified in the step definition
# JUNIT_TEMP_DIR need to be clarified in the step definition
# there is usage accepts a parameter to override the default usage
generate_junit () {
  usage="e2e"
  if [ ! -z $1 ];then
    usage="$1"
  fi
  JUNIT_TEMP_DIR=$(mktemp -d)
  junit_file_name="junit-$usage"
  if [ ! -z "${TEST_PROFILE}" ]; then
    junit_file_name="${junit_file_name}-${TEST_PROFILE}"
  fi
  if [ ! -z "${RUN_TIME}" ]; then
    junit_file_name="${junit_file_name}-${RUN_TIME}"
  fi

  JUNIT_XML="${JUNIT_TEMP_DIR}/${junit_file_name}.xml"

  echo "[CI] the junit temp dir is $JUNIT_TEMP_DIR and junit file will be $JUNIT_XML" 
}

# extract_existing_junit is used to extract existing junit file from previous steps
# this function is used for testing result report step
# It will extract all of the tar.gz files started with junit-
# usage: extract_existing_junit <SHARED_DIR>
extract_existing_junit () {
  if [[ -z "$1" ]];then
    echo "ERROR: at least 1 parameter of uploaded dir is required to scan"
    exit 1
  fi
  uploadedDir=$1
  for file in $(find $uploadedDir -type f -name "junit-*.tar.gz" -maxdepth 1)
  do
    tar -xvf $file -C $uploadedDir
  done
}

# generate_running_cmd is used to generate rosatest running command
# TEST_TIMEOUT is step env variable used to define the timeout of the test run
# JUNIT_XML should be generated by function generate_junit
generate_running_cmd () {
  if [ -z $JUNIT_XML ]; then
    echo "JUNIT_XML is empty, please define it and call generate_junit to generate the value"
    exit 1
  fi 
  if [ -z $TEST_TIMEOUT ]; then
    # set a default value in case it is empty
    TEST_TIMEOUT="4h"
  fi
  cmd="rosatest --ginkgo.v --ginkgo.no-color \
  --ginkgo.timeout ${TEST_TIMEOUT} \
  --ginkgo.junit-report $JUNIT_XML \
  ${LABEL_FILTER_SWITCH}"
  if [ ! -z ${FOCUS} ]; then
    cmd="rosatest --ginkgo.v --ginkgo.no-color \
    --ginkgo.timeout ${TEST_TIMEOUT} \
    --ginkgo.junit-report $JUNIT_XML \
    ${LABEL_FILTER_SWITCH} \
    --ginkgo.focus '${FOCUS}'"
  fi
  echo "$cmd"
}

# upload_junit_result is used to upload the testing result and archive
# first parameter is required to identify which junit file should be uploaded
# second parameter is required to upload the testing result
# third paratemer is required to archive the testing result
# This function can only be called after the cmd running finished which is generated by generate_running_cmd
# usage: upload_junit_result <junit file path> <SHARED_DIR> <ARCHIVE_DIR>
upload_junit_result () {
  echo "[CI] tar and uploading the the testing result"
  # tar and upload the junit.xml files
  if [[ -z "$1" ]]|| [[ -z "$2" ]]|| [[ -z "$3" ]]; then
    echo "ERROR: the usage should be upload_junit_result <junit file path> <SHARED_DIR> <ARCHIVE_DIR>"
    exit 1
  fi
  filePath=$1
  uploadDir=$2
  filename=$(echo "${filePath%.*}" | awk -F "/" '{print $NF}')
  tarPath=${uploadDir}/${filename}.tar.gz
  echo "[CI] going to zip the junit file $filePath to $tarPath"
  tar -zcvf $tarPath $filePath
  echo "[CI] archiving the the testing result"
  # copy the junit.tar.gz to ARTIFACT_DIR
  archiveDir=$3
  cp $tarPath ${archiveDir}
}