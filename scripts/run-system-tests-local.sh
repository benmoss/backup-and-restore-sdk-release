#!/bin/bash -eu

export BOSH_URL="https://genesis-bosh.backup-and-restore.cf-app.com:25555"
export BOSH_CERT_PATH="~/workspace/bosh-backup-and-restore-meta/certs/genesis-bosh.backup-and-restore.cf-app.com.crt"
export BOSH_CLIENT="admin"
export BOSH_CLIENT_SECRET=$(lpass show "GenesisBoshDirectorGCP" --password)
export BOSH_GATEWAY_USER=vcap
export BOSH_GATEWAY_HOST=genesis-bosh.backup-and-restore.cf-app.com
export BOSH_GATEWAY_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem
export POSTGRES_PASSWORD=foo


pushd $(dirname $0)/..
  ginkgo system-tests -trace
popd