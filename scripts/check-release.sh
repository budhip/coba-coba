#!/bin/bash

new_version="$(cat ./semantic_version.txt)"
if [ -z "$new_version" ]; then
    echo "No new version generated. Please use semantic versioning in your commit messages"
    retVal=1
else
    echo "New version '$new_version' generated"
    retVal=0
fi
exit $retVal
