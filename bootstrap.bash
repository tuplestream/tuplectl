#!/usr/bin/env bash
#
# Tuplestream bootstrap script vAUTOREPLACED-VERSION
# Problems? Suggestions? Send a PR to github.com/tuplstream/tuplectl
#
set -euf -o pipefail

setup_tuplectl() {
  echo "Couldn't find tuplectl on your path, "
  OS=$(uname -s)
  
}

command -v tuplectl >/dev/null 2>&1 || { setup_tuplectl; }
