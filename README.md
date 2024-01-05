Simplified nc (netcat), tcp only

Connect stdin, stdout to an open tcp connection's output/input

To build
```
go build -o nc

or 

make
```

To run
```
./nc
```
Example 1:

Transfer a directory from system zos-A to system zos-B
On zos-A (as server)
```
/bin/pax -w -z -x pax mysource/ | nc -l 4321
```
On zos-B (as client)
```
nc zos-A 4321 | /bin/pax -v -ppx -r
```

Example 2:

Transfer a directory from a PC to system zos-A

On zos-A (as server)
```
nc -l 4321 | /bin/iconv -f 1047  -t 819 |  /bin/pax -v -ppx -r
```

On PC (as client)
```
tar -cvf - mydir | nc zos-A 4321
```

Example 3:

Transfer a directory from system zos-A to system zos-B
On zos-A (as server), add data encryption.
```
/bin/pax -w -z -x pax mysource/ | gpg -c --batch --passphrase 123456 -o - 2>/dev/null | nc -l 4321
```
On zos-B (as client)
```
nc zos-A 4321 | gpg -d --batch --passphrase 123456 2>/dev/null | /bin/pax -v -ppx -r
```
