#!/bin/sh

echo "${GCP_SERVICE_ACCOUNT}" | base64 -d > "./${project_id}.json"

echo "authenticating to gcp"
gcloud auth activate-service-account --key-file "./${project_id}.json"
gcloud config set project "${project_id}"
