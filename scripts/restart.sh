#!/bin/sh

echo "redeploying ${service_executable_and_user} with image version: ${VERSION}"

# Cloud Run configuration
gcloud beta config set run/region "${REGION}"
gcloud beta config set run/platform managed

service_manifest_file="deployments/service-${environment}.yaml"
policy_manifest_file="deployments/policy-${environment}.yaml"
IMAGE_NAME="asia-southeast2-docker.pkg.dev/amartha-dev/docker/${service_executable_and_user}"

REVISION=""
 
gen_new_revision() {
    # create 5 random char as suffix; 5 because that the number of character  
    # usually created by the cloud run UI
    # the maximum number of character of revision name is 63, please remember this
    RAND=$( cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 5 | head -n 1)
    REVISION="${service_executable_and_user}-${RAND}"
}

if [ "$1" == "yes" ] 
then 
    gen_new_revision
    # set revision in service manifest yaml
    sed -i "s/__REVISION__/${REVISION}/g" "${service_manifest_file}"
    echo "creating new revision: ${REVISION}"
else 
    # remove revision line from service manifest yaml
    sed -i "s/name: __REVISION__//g" "${service_manifest_file}"
fi

# set image tag in yaml
sed -i "s/__IMAGE_TAG__/${VERSION}/g" "${service_manifest_file}"

# deploy to cloud run
gcloud beta run services replace "${service_manifest_file}"
gcloud beta run services set-iam-policy "${service_executable_and_user}" "${policy_manifest_file}"
