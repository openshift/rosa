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
