#!/bin/sh

echo "Deploying in '${project_id}' project in GCP"
echo "${GCP_SERVICE_ACCOUNT}" | base64 -d > "./${project_id}.json"
VERSION=$(head -1 ./semantic_version.txt)
service_account_email=$(echo "${GCP_SERVICE_ACCOUNT}" | base64 -d | jq .client_email | sed 's/"//g')
image_name="asia-southeast2-docker.pkg.dev/${project_id}/docker/${service_executable_and_user}:${VERSION}"
service_manifest_file="deployments/service-${environment}.yaml"
policy_manifest_file="deployments/policy-${environment}.yaml"

echo "Configuring docker auth for Google Container Registry"
gcloud auth activate-service-account --key-file "./${project_id}.json"
gcloud config set project "${project_id}"
gcloud auth configure-docker asia-southeast2-docker.pkg.dev
gcloud auth print-access-token | docker login -u oauth2accesstoken --password-stdin https://asia-southeast2-docker.pkg.dev

echo "Building image: ${image_name}"
docker build -t "${image_name}" -f ./builds/Dockerfile .
docker push "${image_name}"

# Cloud Run configuration
gcloud beta config set run/region "${REGION}"
gcloud beta config set run/platform managed

# set image tag in yaml
sed -i "s/__IMAGE_TAG__/${VERSION}/g" "${service_manifest_file}"

# remove revision line from the service
sed -i "s/name: __REVISION__//g" "${service_manifest_file}"

# deploy to cloud run
gcloud beta run services replace "${service_manifest_file}"
gcloud beta run services set-iam-policy "${service_executable_and_user}" "${policy_manifest_file}"

# set image tag in yaml
sed -i "s/__IMAGE_TAG__/${VERSION}/g" "${service_manifest_file}"
# deploy to cloud run
gcloud beta run services replace "${service_manifest_file}"
gcloud beta run services set-iam-policy "${service_executable_and_user}" "${policy_manifest_file}"
gcloud run services update-traffic ${service_executable_and_user} --to-latest