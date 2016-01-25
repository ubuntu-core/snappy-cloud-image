#!/bin/sh
append_coverage() {
    local profile="$1"
    if [ -f $profile ]; then
        cat $profile | grep -v "mode: count" >> coverage.out
        rm $profile
    fi
}

echo "mode: count" > coverage.out

for pkg in $(go list ./pkg/...); do
    go test -covermode=count -coverprofile=profile.out "$pkg"
    append_coverage profile.out
done
