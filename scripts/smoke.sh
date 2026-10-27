#!/usr/bin/env bash
# End-to-end smoke check of the own AlbumsService: builds the binaries into bin/, starts the
# server on 127.0.0.1:50061 (an IP literal, so no DNS lookup is on the way), runs the unary
# scenarios and the server stream with the Go client, and stops the server. Needs: go, bash.
set -u
cd "$(dirname "$0")/.."
mkdir -p bin
go build -o bin/ ./cmd/albums-server ./cmd/albums-client || exit 1

ADDR=127.0.0.1:50061
bin/albums-server -addr "$ADDR" > bin/server.log 2>&1 & SRV=$!
trap 'kill $SRV 2>/dev/null' EXIT
for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
  grep -q "albums server started" bin/server.log 2>/dev/null && break
  sleep 0.25
done

C="bin/albums-client -addr $ADDR"
fail=0
check() { # check "description" "expected substring" command...
  local name=$1 want=$2; shift 2
  local out; out=$("$@" 2>&1)
  if grep -q -- "$want" <<<"$out"; then echo "  OK   $name"; else echo "  FAIL $name"; echo "$out" | head -3; fail=1; fi
}

echo "unary:"
check "GetAlbum 1 returns Kind of Blue"                 '"Kind of Blue"'          $C get 1
check "GetAlbum 999 -> NotFound"                        'code=NotFound'           $C get 999
check "GetAlbum 0 -> InvalidArgument"                   'code=InvalidArgument'    $C get 0
check "CreateAlbum -> id 11"                            '"11"'                    $C create "Blue Train" "Blue Note" 1 2
check "CreateAlbum duplicate -> AlreadyExists"          'code=AlreadyExists'      $C create "Blue Train" "Blue Note" 1 2
check "CreateAlbum unknown genre -> FailedPrecondition" 'code=FailedPrecondition' $C create "Giant Steps" "Atlantic" 99 2
check "CreateAlbum without title -> InvalidArgument"    'code=InvalidArgument'    $C create "" "Atlantic" 1 2

echo "server streaming:"
$C -max 2 watch > bin/watch.out 2>&1 & W=$!
for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
  grep -q "watch started" bin/server.log 2>/dev/null && break
  sleep 0.25
done
$C create "Giant Steps" "Atlantic" 1 2 > /dev/null 2>&1
$C create "My Favorite Things" "Atlantic" 1 2 > /dev/null 2>&1
wait $W
events=$(grep -c "CREATED" bin/watch.out)
if [ "$events" -eq 2 ]; then echo "  OK   WatchAlbums delivers the 2 CREATED events"; else echo "  FAIL WatchAlbums delivers the 2 CREATED events (got $events)"; head -5 bin/watch.out; fail=1; fi
check "WatchAlbums ends with io.EOF after max_events" 'io.EOF' cat bin/watch.out
check "server log: watch finished, max_events reached" 'max_events reached' cat bin/server.log

[ $fail -eq 0 ] && echo "ALL OK" || { echo "FAILURES"; exit 1; }
