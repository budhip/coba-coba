#!/bin/bash
#********************************************************************************
# PROGRAM       :  generate-mock.sh
# DESCRIPTION   :  Generate mock files find in specified folder and generate all
# DATE          :  1 Feb 2023
# ORIGINATOR    :  Syldie
# RUN           :  ./generate-mock.sh service
#********************************************************************************

if [[ $1 != "" ]]; then

  for d in $(find . -path '*/'"$1"'/*' -not -path '*/mock/*' ! -path '*_test.go' ! -path '*_mock.go' ! -path '*base.go' ! -path '*query.go'); do
    fileName="${d##*/}"
    dir="$(dirname "$d")"

    if [[ $fileName != "mock" ]]; then
      name=${fileName%.*}
      fileGen="${name}_mock.go"
      destination="${dir}/mock"

      if grep -E "type [[:alnum:]]+ interface" "$dir/$fileName" > /dev/null; then
        printf "generating mock for $fileGen ..."
        mockgen -source="$dir/$fileName" -destination="$destination/$fileGen" -package=mock
        printf "finished\n"
      else
        printf "no interface found in $dir/$fileName, no mock generated ...\n"
      fi
    fi

  done
  printf "mock $1 successfully generated ...\n"

else

  printf "directory must be filled, no mock generated ...\n"

fi
