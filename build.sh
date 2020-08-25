#!/usr/bin/env bash
set -euo pipefail

REPO=kumorilabs

push=0
while [ ! $# -eq 0 ]
do
    case "$1" in
        --push | -p)
            push=1
            ;;
    esac
    shift
done

for d in ./*
do
    if [ -d "$d" ]; then
        fn=$(basename $d)

        if [ "$fn" == "bin" ]; then
            continue
        fi

        ver=$(cat $d/VERSION)
        docker build -t $REPO/kpt-fn-$fn:$ver -f ./Dockerfile --build-arg=FN=$fn .
        if [ $push -eq 1 ]; then
            docker push $REPO/fn-$fn:$ver
        fi
    fi
done
