#!/bin/bash
set -e

runner::configure() {
  echo "Configuring Github Action Runner ..."
  if [ ! -z "$RUNNER_JITCONFIG" ]; then
    echo "Github Action Runner jitconfig detected. No need to configure ..."
    return 0
  fi 

  if [ -z "$URL" ]; then
    echo 1>&2 "error: missing URL environment variable"
    exit 1
  fi

  RUNNER_TOKEN_FILE=${RUNNER_TOKEN_FILE:-.token}
  if [ ! -f "$RUNNER_TOKEN_FILE" ]; then
    if [ -z "$RUNNER_TOKEN" ]; then
      echo 1>&2 "error: missing RUNNER_TOKEN environment variable"
      exit 1
    fi

    echo -n $RUNNER_TOKEN > "$RUNNER_TOKEN_FILE"
  fi

  unset RUNNER_TOKEN
  source /runner/env.sh
  /runner/config.sh --unattended \
    --url "${URL}" \
    --name "${RUNNER_NAME}" \
    --token $(cat "$RUNNER_TOKEN_FILE") \
    --labels "${RUNNER_LABELS}" \
    --runnergroup "${RUNNER_GROUP:-Default}" \
    --work "${RUNNER_WORKDIR:-_work}" \
    --replace --ephemeral --disableupdate & wait $!

  echo "Configured Github Action Runner ..."
}

runner::run() {
  echo "Running Github Action Runner ..."
  if [ ! -z "$RUNNER_JITCONFIG" ]; then
    /runner/run.sh --jitconfig "${RUNNER_JITCONFIG}"
  else
    trap 'runner::cleanup; exit 130' INT
    trap 'runner::cleanup; exit 143' TERM
    /runner/run.sh "$@" & wait $!
  fi 
}

runner::cleanup() {
  if [ ! -z "$RUNNER_JITCONFIG" ]; then
    echo "Github Action Runner jitconfig detected. No need to cleanup ..."
    return 0
  fi 

  if [ -f .runner ]; then
    echo "Teardown. Github Action Runner ..."
    while true; do
      ./config.sh remove --token $(cat "$RUNNER_TOKEN_FILE") && break
      echo "Retrying in 30 seconds..."
      sleep 30
    done
  fi 
}

dockerd_pid=/run/user/"$(id -u)"/docker.pid
dockerd_sock=/run/user/"$(id -u)"/docker.sock
dockerd::start() {
  export DOCKER_HOST=unix://$dockerd_sock
  dockerd-entrypoint.sh dockerd &

  retry=5
  while true; do
    docker ps -q > /dev/null 2>&1 && break
    if [[ $retry -le 0 ]]; then
      echo "Reached maximum attempts, not waiting any longer..."
      exit 1
    fi

    echo "Waiting for docker to be ready, sleeping for 5 seconds."
    retry=$((retry-1)) 
    sleep 5
  done
}

dockerd::shutdown() {
  kill -s SIGINT $(cat $dockerd_pid)

  retry=5
  while true; do
    ! $(docker ps -q > /dev/null 2>&1) && break
    if [[ $retry -le 0 ]]; then
      echo "Reached maximum attempts, not waiting any longer..."
      exit 1
    fi

    echo "Waiting for docker to be shutdown, sleeping for 5 seconds."
    retry=$((retry-1)) 
    sleep 5
  done
}

runner::configure
if type dockerd-entrypoint.sh &>/dev/null; then
  echo "Starting Docker Daemon ..."
  dockerd::start
fi

runner::run
if [ -f $dockerd_pid ]; then
  echo "Shutdown Docker Daemon ..."
  dockerd::shutdown
fi
