#! /usr/bin/bash

script_dir=$(dirname $0)

cp $script_dir/og_tests/* $script_dir/../testi

pushd $script_dir/../testi

zip testi.zip * && rm kp.*

popd