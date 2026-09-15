#!/usr/bin/env bash
# bench-compare runs the benchmarks of BASE and of the working tree in interleaved rounds and compares
# them with benchstat. `make bench-compare` runs it.
#
#   BASE       the commit to compare against, used as given    (default origin/master)
#   ROUNDS     rounds; each runs both sides, alternating first  (default 6)
#   BENCH      go test -bench pattern                           (default .)
#   BENCHTIME  go test -benchtime                               (default 400ms)
#
# Both sides are built with the working tree's benchmark_test.go and no other test file, so they run
# the same cases. The raw results are kept under .bench-compare/.
#
# Written for bash 3.2, which macOS ships.
set -euo pipefail

benchstat=golang.org/x/perf/cmd/benchstat@v0.0.0-20260908200009-22c9c6c9d4da

base=${BASE:-origin/master}
rounds=${ROUNDS:-6}
bench=${BENCH:-.}
benchtime=${BENCHTIME:-400ms}

die() {
	echo "bench-compare: $*" >&2
	exit 1
}

case $rounds in
'' | *[!0-9]*) die "ROUNDS must be a positive integer, not '$rounds'" ;;
esac
[ "$rounds" -gt 0 ] || die "ROUNDS must be a positive integer, not '$rounds'"

cd "$(git rev-parse --show-toplevel)"
[ -f benchmark_test.go ] || die "no benchmark_test.go in $(pwd)"

sha=$(git rev-parse --verify --quiet "$base^{commit}") ||
	die "BASE=$base is not a commit; run git fetch, or pass BASE=<ref>"
git merge-base --is-ancestor "$sha" HEAD ||
	echo "bench-compare: warning: $base is not an ancestor of HEAD, so its own commits count as changes too; BASE=\$(git merge-base origin/master HEAD) compares against the fork point" >&2

work=$(mktemp -d "${TMPDIR:-/tmp}/bench-compare.XXXXXX")
trap 'rm -rf "$work"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# The base tree is BASE's commit. The head tree is every file git would commit, as it is on disk:
# uncommitted edits and untracked files that are not ignored are in, deleted files are out.
mkdir "$work/base" "$work/head"
git archive "$sha" | tar -x -C "$work/base"
git ls-files -z --cached --others --exclude-standard | while IFS= read -r -d '' f; do
	[ -f "$f" ] || continue
	mkdir -p "$work/head/$(dirname "$f")"
	cp "$f" "$work/head/$f"
done
for side in base head; do
	find "$work/$side" -name '*_test.go' -delete
	cp benchmark_test.go "$work/$side/benchmark_test.go"
done

# -trimpath gives both binaries the same file names, which would otherwise differ by tree.
build() {
	(cd "$work/$1" && go test -c -trimpath -o "$work/$1.test" .)
}
build base || die "BASE=$base ($sha) does not build with the working tree's benchmark_test.go; the compiler's errors are above"
build head || die "the working tree does not build; the compiler's errors are above"

base_go=$(go version "$work/base.test" | awk '{ print $NF }')
head_go=$(go version "$work/head.test" | awk '{ print $NF }')
[ "$base_go" = "$head_go" ] || die "base was built with $base_go and head with $head_go; set GOTOOLCHAIN so that both use one toolchain"

mkdir -p .bench-compare
out=$(mktemp -d ".bench-compare/$(date -u +%Y%m%dT%H%M%SZ).XXXX")

dirty=
[ -z "$(git status --porcelain)" ] || dirty=', with uncommitted changes'
# No "key: value" lines go into base.txt or head.txt: benchstat would split its tables on a key whose
# value differs between them.
{
	echo "base:      $base = $(git log -1 --format='%h %s' "$sha")"
	echo "head:      working tree on $(git log -1 --format='%h %s' HEAD)$dirty"
	echo "go:        $head_go, $(uname -sm)"
	echo "rounds:    $rounds"
	echo "bench:     $bench"
	echo "benchtime: $benchtime"
} | tee "$out/info.txt"
if [ -n "$dirty" ]; then
	git status --short --untracked-files=all >>"$out/info.txt"
	# head.diff rebuilds the head tree from HEAD, untracked files included. It is taken through a copy
	# of the index and a scratch object directory, so neither the real index nor the repository's
	# objects change.
	objects=$(cd "$(git rev-parse --git-path objects)" && pwd)
	mkdir "$work/objects"
	index=$(git rev-parse --git-path index)
	[ ! -f "$index" ] || cp "$index" "$work/index"
	GIT_INDEX_FILE="$work/index" GIT_OBJECT_DIRECTORY="$work/objects" GIT_ALTERNATE_OBJECT_DIRECTORIES="$objects" \
		sh -c 'git add -A && git diff --no-ext-diff --no-color --binary --cached HEAD' >"$out/head.diff"
fi

# Alternating which side runs first spreads warm-up and drift over both sides alike.
for round in $(seq 1 "$rounds"); do
	order="base head"
	[ $((round % 2)) -eq 1 ] || order="head base"
	for side in $order; do
		echo "bench-compare: round $round/$rounds, $side" >&2
		if ! "$work/$side.test" -test.run '^$' -test.bench "$bench" -test.benchtime "$benchtime" -test.benchmem >>"$out/$side.txt"; then
			# A failure in the first round, such as an invalid BENCH or BENCHTIME, leaves nothing worth
			# keeping, so its output is shown and the directory goes, as it does when BENCH matches nothing.
			if [ "$round" -eq 1 ]; then
				cat "$out/$side.txt" >&2
				rm -rf "$out"
				die "$side failed in round 1; its output is above"
			fi
			die "$side failed in round $round; its output ends $out/$side.txt"
		fi
	done
	if [ "$round" -eq 1 ] && ! grep -q '^Benchmark' "$out/head.txt"; then
		rm -rf "$out"
		die "BENCH=$bench matches no benchmark"
	fi
done

status=0
if (cd "$out" && go run "$benchstat" base.txt head.txt) >"$out/benchstat.txt"; then
	cat "$out/benchstat.txt"
else
	echo "bench-compare: benchstat failed; it needs Go 1.26 or later, which GOTOOLCHAIN=local can rule out" >&2
	status=1
fi
echo "bench-compare: raw results in $(pwd)/$out; compare them again there with: go run $benchstat base.txt head.txt" >&2
exit "$status"
