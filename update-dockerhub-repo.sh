#!/bin/bash
set -e

# Json file containing the description of the repository
INPUT_JSON_FILE=$1

if [ ! -f ${INPUT_JSON_FILE} ]; then
    echo "File '${INPUT_JSON_FILE}' is missing. Exiting now..."
    exit
fi

# DOCKERHUB_CREDS_USR and DOCKERHUB_CREDS_PSW must be present as environment variables
# DOCKERHUB_CREDS_USR will typically be 'docker'
if [ -z "$DOCKERHUB_CREDS_USR" ] || [ -z "$DOCKERHUB_CREDS_PSW" ] ; then
    echo "Variables missing. Seems that you are running from a console. Prompting user and passwd"
    read  -p 'Username: ' DOCKERHUB_CREDS_USR
    read -sp 'Password: ' DOCKERHUB_CREDS_PSW
    echo
fi

# Read Json file with data about the repository
DOCKERHUB_REPO=`cat ${INPUT_JSON_FILE}`
DESCRIPTION_SHORT=$(echo ${DOCKERHUB_REPO} | jq -r '.repository.description')
DESCRIPTION_FULL=$(echo ${DOCKERHUB_REPO} | jq -r '.repository.full_description')

# Get JWT token
echo
echo "Getting token"
TOKEN=$(curl -v -XPOST \
    -H "Content-Type: application/json" \
    -d '{"username": "'${DOCKERHUB_CREDS_USR}'", "password": "'${DOCKERHUB_CREDS_PSW}'"}' \
    https://hub.docker.com/v2/users/login/ | jq -r '.token')

# Update
echo
echo "Updating"
curl -v -XPATCH \
    -H "Content-Type: application/json" \
    -H "Authorization: JWT ${TOKEN}" \
    -d "{\"description\": \"${DESCRIPTION_SHORT}\", \"full_description\": \"${DESCRIPTION_FULL}\"}" \
    https://hub.docker.com/v2/repositories/docker/app/

# Logout
echo
echo "Logging out"
curl -v -XPOST \
    -H "Accept: application/json" \
    -H "Authorization: JWT ${TOKEN}" \
    https://hub.docker.com/v2/logout/
