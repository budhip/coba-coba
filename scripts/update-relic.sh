VERSION=$(head -1 ./semantic_version.txt)

export commit_message=$(git log -n 1 --pretty="format:%s ${BITBUCKET_COMMIT}")
export commit_user=$(git log -n 1 --pretty="format:%an ${BITBUCKET_COMMIT}")

echo "{\"deployment\":{\"revision\":\"${VERSION}\",\"changelog\":\"${commit_message}\",\"user\":\"${commit_user}\"}}" > new_relic.txt

curl -v -X POST "https://api.newrelic.com/v2/applications/${RELIC_APP_ID}/deployments.json" \
    -H "X-Api-Key: ${RELIC_API_KEY}" -i -H "Content-Type: application/json" \
    -d "$(sed ':a;N;$!ba;s/\n/ /g' ./new_relic.txt)"