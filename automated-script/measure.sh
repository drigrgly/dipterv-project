#!/bin/bash

#------
# VARS
#------

# General
CLUSTERS=()

# Iperf / Load Generator

#iperf -c localhost -p 5000 -u -i 1 -l 100 -b 800000 -t 10 -e | tee -a $log_file

#-------------------
# Reading arguments
#-------------------
# More safety, by turning some bugs into errors.
set -o errexit -o pipefail -o noclobber -o nounset

# ignore errexit with `&& true`
getopt --test > /dev/null && true
if [[ $? -ne 4 ]]; then
    echo '`getopt --test` failed in this environment.'
    exit 1
fi

# option --output/-o requires 1 argument
LONGOPTS=debug,cluster:,output:,verbose
OPTIONS=dfc:o:v

# -temporarily store output to be able to check for errors
# -activate quoting/enhanced mode (e.g. by writing out “--options”)
# -pass arguments only via   -- "$@"   to separate them correctly
# -if getopt fails, it complains itself to stderr
PARSED=$(getopt --options=$OPTIONS --longoptions=$LONGOPTS --name "$0" -- "$@") || exit 2
# read getopt’s output this way to handle the quoting right:
eval set -- "$PARSED"

d=n f=n v=n outFile=-
# now enjoy the options in order and nicely split until we see --
while true; do
    case "$1" in
        -d|--debug)
            d=y
            shift
            ;;
        -f|--force)
            f=y
            shift
            ;;
        -c|--cluster)
            CLUSTERS+=("$2")
            shift 2
            ;;
        -v|--verbose)
            v=y
            shift
            ;;
        -o|--output)
            outFile="$2"
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "Programming error"
            exit 3
            ;;
    esac
done

# handle non-option arguments
if [[ $# -ne 1 ]]; then
    echo "$0: A single input file is required."
    exit 4
fi

#-------------------
# Start measurement 
#-------------------

# Create a name for the log file based on the current date and time
log_file="iperf_client_$(date +%Y%m%d_%H%M%S).log"

# Store the beginning time
start_time=$(date +%s)
echo $start_time > $log_file

# Start turncat in the background and store the pid
turncat --log=all:INFO udp://127.0.0.1:5000 k8s://stunner/udp-gateway:udp-listener udp://$(kubectl get svc iperf-server -o jsonpath="{.spec.clusterIP}"):5001 &
tc_pid=$!

echo "Running"

# Run iperf in client mode to send UDP traffic to the turncat server
iperf -c localhost -p 5000 -u -i 1 -l 100 -b 800000 -t 10 -e | tee -a $log_file

# Store the ending time
end_time=$(date +%s)

echo $end_time >> $log_file

# Terminate turncat after the test is complete
kill $tc_pid