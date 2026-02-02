# Author: Hunter Kepley
# 2026 
# Red Hat
#
## This is meant to take in the output of ./hack/release-list-jiras.sh
## and turn it into a usable output for Errata. This script saves quite a 
## bit of time every single release manually fixing them

a = input("Paste JIRAs received from './hack/release-list-jiras.sh' directly here: ")

print("Removing extra whitespace")

b = a.strip().split()

print("Removing duplicates")

c = list(set([x.lower() for x in b]))

print("Capitalizing")

final = ""

for k in c:
    final += k + " "

print("\n\n--------------------\n\n" + final.strip().upper())

