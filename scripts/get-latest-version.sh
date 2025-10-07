#!/bin/sh

LATEST_TAG=""
IMAGE_NAME="asia.gcr.io/${project_id}/${service_executable_and_user}"
get_latest_tag () {
    RES=$(gcloud container images list-tags $IMAGE_NAME --format='get(tags)')
    while IFS=' ' read -ra TAGS; do
        LATEST_TAG="${TAGS[0]}"
        break
    done <<< "$RES"
}

get_latest_tag
export VERSION=${LATEST_TAG}
