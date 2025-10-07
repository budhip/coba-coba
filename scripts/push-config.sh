#!/bin/sh -ex
# host = Tower host
# username = Tower username
# password = Tower user password
# ID = ID of job template being launched.

echo "Configuring Tower Settings"
export TOWER_HOST=$tower_host
export TOWER_USERNAME=$tower_username
export TOWER_PASSWORD=$tower_password

if [[ $TOWER_USERNAME == "" ]] || [[ $TOWER_PASSWORD  == "" ]]
then
  echo "-- WARNING: Configuration has not been fully set -";
  echo "---- Set TOWER_HOST, TOWER_USERNAME, and TOWER_PASSWORD";
  echo "---- environment variables first";
fi


config_file="config/${config_name}"
consulconfigfile=$(cat ${config_file} | base64)

# Deploy the service
awx -k job_templates launch push-config-consul-2 --monitor --extra_vars "{
  \"consul_host\": \"$consul_host\",
  \"vault_host\": \"$vault_host\",
  \"config_type\": \"yaml\",
  \"config_name\": \"$config_name\",
  \"repo_name\": \"$service_executable_and_user\",
  \"repo_branch\": \"$repo_branch\",
  \"kv_of\": \"$kv_of\",
  \"service_secret_name\": \"$consul_service_key\",
  \"consul_service_key\": \"$consul_service_key\",
  \"consul_config_file\": \"$consulconfigfile\",
  \"vault_token\": \"$vault_token\",
  \"service_exec_and_user\": \"$service_executable_and_user\"
  }"
