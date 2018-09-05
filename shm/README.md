
# Shared memory speed test

This tests passing large requests via SystemV-style shared memory vs. localhost
HTTP.

## shm settings

First ensure that your shm limits are high enough:

```
$ sudo sysctl -w kern.sysv | grep shm
kern.sysv.shmmax: 134217728
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

```$ sudo sysctl -w kern.sysv.shmmax=134217728```

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

```
$ go run client.go

HTTP:
...
Time: 34139065000 ns; qps: 29.291956

SHM:
...
Time: 32949964000 ns; qps: 30.349047
```

You can also set the number of threads, total requests, and bytes/request:
```
$ go run client.go --nt=15 --nr=1000 --bytes=33554432
```

To see all existing shm segments:
```
$ ipcs -a
```

And if you accidentally crash the server/client and need to clean up segments:
```
$ ipcs -a | cut -d ' ' -f2 | xargs -n 1 ipcrm -m
```

