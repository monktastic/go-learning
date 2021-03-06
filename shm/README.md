
# Shared memory speed test

This tests passing large requests via SystemV-style shared memory vs. localhost
HTTP.

## shm settings

First ensure that your shm limits are high enough:

```
$ sudo sysctl -w kern.sysv | grep shm
kern.sysv.shmmax=1073741824
kern.sysv.shmmin: 1
kern.sysv.shmmni: 1024
kern.sysv.shmseg: 1024
kern.sysv.shmall: 262144
```

* `shmmin` is min bytes per segment.
* `shmmax` is max bytes per segment.
* `shmall` is max shared pages on the system. Multiply by `PAGE_SIZE` (`getconf PAGE_SIZE`) to get max bytes total.
* `shmmni` is max segments total.
* `shmseg` is max segments attached per process.

They can be changed with `sysctl`:

```$ sudo sysctl -w kern.sysv.shmmax=1073741824```

Or by editing `/etc/sysctl.conf`.

### Go deps

```
$ go get github.com/gen2brain/shm
```

---

## Running the test

```
$ go run server.go
```


The client uses 15 threads, sends 1000 total requests, each 2^25 bytes long:
```
$ go run client.go
1000 15 33554432
HTTP:
...
Time: 99952930000 ns; qps: 10.004709

SHM:
...
Time: 37719854000 ns; qps: 26.511237
```

You can also pass flags:
```
$ go run client.go -nt=4 -nr=100 -bytes=128000000
100 4 128000000
HTTP:
...
Time: 34616928000 ns; qps: 2.888760
SHM:
...
Time: 12129793000 ns; qps: 8.244164
```

To see all existing shm segments:
```
$ ipcs -a
```

And if you accidentally crash the server/client and need to clean up segments:
```
$ ipcs -a | cut -d ' ' -f2 | xargs -n 1 ipcrm -m
```

