#!/bin/bash
set -e

REPOS_DIR="/home/cova/repos"

mkdir -p "$REPOS_DIR"
for src in /testdata/*/; do
    name=$(basename "$src")
    dest="$REPOS_DIR/$name"
    cp -r "$src" "$dest"
    git -C "$dest" init -q
    git -C "$dest" -c user.email=sandbox@test -c user.name=sandbox \
        -c commit.gpgsign=false add -A
    git -C "$dest" -c user.email=sandbox@test -c user.name=sandbox \
        -c commit.gpgsign=false commit -q -m init
done

chown -R cova:cova /home/cova

echo "cova sandbox"
echo ""
echo "sample covens:"
echo "  single: file://$REPOS_DIR/coven"
echo "  multi:  file://$REPOS_DIR/multicoven"
echo ""
echo "type 'exit' to discard everything"

exec su - cova
