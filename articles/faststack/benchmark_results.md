# benchmark results

all benchmark results are on a AMD Ryzen 9 5900 12-Core Processor @ ~3GHz with 32GiB of RAM.

## linux(wsl): SSD

BenchmarkPing/depth=1/faststack-24 14306 78653 ns/op 8849 B/op 92 allocs/op
BenchmarkPing/depth=1/slowstack-24 12033 105193 ns/op 220315 B/op 101 allocs/op
BenchmarkPing/depth=5/faststack-24 10000 101501 ns/op 10835 B/op 132 allocs/op
BenchmarkPing/depth=5/slowstack-24 7717 135690 ns/op 302127 B/op 157 allocs/op
BenchmarkPing/depth=25/faststack-24 6556 180071 ns/op 20765 B/op 332 allocs/op
BenchmarkPing/depth=25/slowstack-24 3907 313058 ns/op 719256 B/op 440 allocs/op
BenchmarkPing/depth=125/faststack-24 4456 260204 ns/op 44900 B/op 651 allocs/op
BenchmarkPing/depth=125/slowstack-24 921 1341206 ns/op 2797236 B/op 1846 allocs/op
BenchmarkPing/depth=625/faststack-24 3972 264099 ns/op 44901 B/op 651 allocs/op
BenchmarkPing/depth=625/slowstack-24 100 10104571 ns/op 13171325 B/op 8863 allocs/op
BenchmarkPing/depth=3125/faststack-24 4537 265868 ns/op 44900 B/op 651 allocs/op
BenchmarkPing/depth=3125/slowstack-24 8 140203673 ns/op 64903352 B/op 43949 allocs/op

## linux(wsl): SSD: trimpath

BenchmarkPing/depth=1/faststack-24 170720 7152 ns/op 5253 B/op 33 allocs/op
BenchmarkPing/depth=1/slowstack-24 84520 16056 ns/op 3659 B/op 56 allocs/op
BenchmarkPing/depth=5/faststack-24 102102 10808 ns/op 5573 B/op 57 allocs/op
BenchmarkPing/depth=5/slowstack-24 43609 27623 ns/op 7375 B/op 105 allocs/op
BenchmarkPing/depth=25/faststack-24 48518 24485 ns/op 7175 B/op 177 allocs/op
BenchmarkPing/depth=25/slowstack-24 9349 116412 ns/op 26979 B/op 347 allocs/op
BenchmarkPing/depth=125/faststack-24 43698 26656 ns/op 7431 B/op 196 allocs/op
BenchmarkPing/depth=125/slowstack-24 1110 1028623 ns/op 118842 B/op 1549 allocs/op
BenchmarkPing/depth=625/faststack-24 36564 33081 ns/op 7431 B/op 196 allocs/op
BenchmarkPing/depth=625/slowstack-24 68 17138684 ns/op 684806 B/op 7553 allocs/op
BenchmarkPing/depth=3125/faststack-24 20023 58641 ns/op 7431 B/op 196 allocs/op
BenchmarkPing/depth=3125/slowstack-24 3 399329411 ns/op 3155112 B/op 37560 allocs/op

## windows: SSD

BenchmarkPing/depth=1/faststack-24 2938 364453 ns/op 14329 B/op 100 allocs/op
BenchmarkPing/depth=1/slowstack-24 2830 384820 ns/op 223090 B/op 105 allocs/op
BenchmarkPing/depth=5/faststack-24 2568 493837 ns/op 18493 B/op 140 allocs/op
BenchmarkPing/depth=5/slowstack-24 2040 594944 ns/op 306681 B/op 161 allocs/op
BenchmarkPing/depth=25/faststack-24 982 1138067 ns/op 39313 B/op 340 allocs/op
BenchmarkPing/depth=25/slowstack-24 738 1602837 ns/op 730882 B/op 444 allocs/op
BenchmarkPing/depth=125/faststack-24 554 2115883 ns/op 80025 B/op 653 allocs/op
BenchmarkPing/depth=125/slowstack-24 176 6788388 ns/op 2845610 B/op 1850 allocs/op
BenchmarkPing/depth=625/faststack-24 542 2154781 ns/op 80032 B/op 653 allocs/op
BenchmarkPing/depth=625/slowstack-24 33 36139921 ns/op 13394896 B/op 8870 allocs/op
BenchmarkPing/depth=3125/faststack-24 513 2236904 ns/op 80033 B/op 653 allocs/op
BenchmarkPing/depth=3125/slowstack-24 4 255713675 ns/op 66048070 B/op 43975 allocs/op

# windows: HDD

goos: windows
goarch: amd64
pkg: gitlab.com/efronlicht/gin-ex
cpu: AMD Ryzen 9 5900 12-Core Processor
BenchmarkPing/depth=1/faststack-24 2294 476571 ns/op 14329 B/op 100 allocs/op
BenchmarkPing/depth=1/slowstack-24 2394 509789 ns/op 223090 B/op 105 allocs/op
BenchmarkPing/depth=5/faststack-24 1602 710711 ns/op 18492 B/op 140 allocs/op
BenchmarkPing/depth=5/slowstack-24 1381 813649 ns/op 306680 B/op 161 allocs/op
BenchmarkPing/depth=25/faststack-24 571 1878803 ns/op 39315 B/op 340 allocs/op
BenchmarkPing/depth=25/slowstack-24 500 2340672 ns/op 730874 B/op 444 allocs/op
BenchmarkPing/depth=125/faststack-24 324 3701402 ns/op 80024 B/op 653 allocs/op
BenchmarkPing/depth=125/slowstack-24 100 10164412 ns/op 2845715 B/op 1850 allocs/op
BenchmarkPing/depth=625/faststack-24 309 3739220 ns/op 80048 B/op 653 allocs/op
BenchmarkPing/depth=625/slowstack-24 22 51105918 ns/op 13394559 B/op 8870 allocs/op
BenchmarkPing/depth=3125/faststack-24 316 3756325 ns/op 80027 B/op 653 allocs/op
BenchmarkPing/depth=3125/slowstack-24 3 334763167 ns/op 66048874 B/op 43968 allocs/op
