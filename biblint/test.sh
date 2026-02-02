#!/bin/bash

# This script will run biblint clean against all the *_in.bib files in
# the tests subdirectory. It should be run with CWD = the location of
# the biblint executable. Output files are placed in ./test_out (they
# are overwritten each time test.sh is run. The expected outputs are
# in tests/*_exp.bib. 

TESTOUTDIR=test_out
mkdir -p "$TESTOUTDIR"

echo "# Running biblint tests..."
echo "# ===================="
echo "#   biblint clean"
echo "# ===================="

for f in tests/clean_*_in.bib ; do
    bn=`basename $f _in.bib`
    exp="tests/${bn}_exp.bib"
    out="$TESTOUTDIR/${bn}_out.bib"

    ./biblint clean -quiet=true --merge-journal-names=2 $f > $out
    if ! cmp -s $exp $out ; then
        echo "FAILED: $bn `cmp $exp $out`"
    else
        echo "PASSED: $bn"
    fi
done

echo "# ===================="
echo "#   biblint check"
echo "# ===================="
for f in tests/check_*_in.bib ; do
    bn=`basename $f _in.bib`
    exp="tests/${bn}_exp.bib"
    out="$TESTOUTDIR/${bn}_out.bib"

    ./biblint check -quiet=true $f > $out
    if ! cmp -s $exp $out ; then
        echo "FAILED: $bn `cmp $exp $out`"
    else
        echo "PASSED: $bn"
    fi
done
