#!/bin/bash
# we want to sync the dev "cdn" bucket to the prod "cdn" bucket

aws s3 sync s3://proglv-public s3://proglv-dev