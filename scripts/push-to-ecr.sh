#!/bin/bash

VERSION=$(head -1 ./semantic_version.txt)
image_name="asia-southeast2-docker.pkg.dev/${project_id}/docker/${service_executable_and_user}:${VERSION}"

pip install awscli

export AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_JKT"
export AWS_SECRET_ACCESS_KEY="$AWS_SECRET_KEY_JKT"
export AWS_DEFAULT_REGION="ap-southeast-3"

aws ecr get-login-password --region $AWS_DEFAULT_REGION | docker login --username AWS --password-stdin $ECR_HOST

p=$(aws ecr describe-repositories --query "repositories[].[repositoryName]" --output text | grep $repo_name | wc -l)

if [[ ${p##*( )} -eq 1 ]]; then
  echo "Repository already exist!"
else
  echo "Creating the repository first."
  aws ecr create-repository --repository-name $project_id/$repo_name
fi

docker images
docker tag ${image_name} $ECR_HOST/$project_id/$repo_name:$VERSION
docker push $ECR_HOST/$project_id/$repo_name:$VERSION
