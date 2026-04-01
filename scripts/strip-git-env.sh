#!/bin/bash
# strip-git-env removes git repository discovery variables from the environment.
#
# When git invokes hooks, it sets GIT_DIR, GIT_WORK_TREE, etc. in the hook
# environment. These leak to child processes (lefthook -> go test -> exec.Command).
# Test helpers that run "git init --bare" or "git clone --bare" in temp dirs can
# then write core.bare=true to the parent repo's .git/config via a race condition
# on shared file descriptors.
#
# Usage: . scripts/strip-git-env.sh && command ...
unset GIT_DIR
unset GIT_WORK_TREE
unset GIT_COMMON_DIR
unset GIT_OBJECT_DIRECTORY
unset GIT_INDEX_FILE
unset GIT_ALTERNATE_OBJECT_DIRECTORIES
