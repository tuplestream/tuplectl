#!/usr/bin/env bash
#
# Tuplestream bootstrap script
# Problems? Suggestions? Send a PR to github.com/tuplstream/tuplectl
# or discuss on Gitter: https://gitter.im/tuplestream/community
#
set -euf -o pipefail

leaving_early() {
  echo "Bye for now 😢"
  exit 1
}

install_tuplectl() {
  chmod +x tuplectl
  sudo mv tuplectl /usr/local/bin/
  echo "$(tuplectl version) installed! 🌈"
}

download_tuplectl() {
  OS=$(uname -s)
  curl -o tuplectl -L https://github.com/tuplestream/tuplectl/releases/latest/download/tuplectl-Darwin-amd64 --progress-bar
  echo "tuplectl will be installed at /usr/local/bin/tuplectl, you'll be prompted for your root password to move it there. Is this ok? [Y/n]"
  read -s -n 1 input
  if [[ $input = "" ]]; then
    install_tuplectl
  elif [[ $input = "Y" ]]; then
    install_tuplectl
  elif [[ $input = "y" ]]; then
    install_tuplectl
  else
    rm -f tuplectl
    leaving_early
  fi

  tuplectl setup
}

setup_tuplectl() {
  echo "Couldn't find tuplectl on your path, download and install now? [Y/n]"
  read -s -n 1 input
  if [[ $input = "" ]]; then 
    download_tuplectl
  elif [[ $input = "Y" ]]; then
    download_tuplectl
  elif [[ $input = "y" ]]; then
    download_tuplectl
  else
    leaving_early
  fi
}

command -v tuplectl >/dev/null 2>&1 || { setup_tuplectl; }
