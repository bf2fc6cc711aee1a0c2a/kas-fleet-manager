#!/bin/bash -e

# First attempt to generate new code, hiding the output of the command as it is soo chatty.
make generate code/fix > /dev/null

# Then make sure that no new code was generated 
changed_file_count=$(git status -s | wc -l)

if [ $changed_file_count -gt 0 ]; then
    git status -s
    echo "Please run 'make generate code/fix' to update generated code and fix formatting. Make sure that the above mentioned files are also commited."
    # Fail the execution of the script
    exit 1
fi

# Exit with success
exit 0