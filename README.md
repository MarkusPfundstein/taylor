# taylor
distributed worker queue written in golang

## Dependencies

- uuid
- gin
- ql

## To-Do

- consul integration
- docker driver
- add maximal job lifetime
- add job.OnError {reschedule, webhook callback, script callback, nothing} (Exec on server)
- add job.OnSuccess {webhook callback, script callback, nothing} (Exec on server)
- add job.OnUpdate {webhook callback, script callback, nothing} (Exec on server)
- add TLS
- -dev mode
- cmd line args
- docs
- remove deletion of database on startup :D :D
