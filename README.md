# taylor
distributed worker queue written in golang

## Dependencies

- uuid
- gin
- ql

## To-Do

- add agent side validation of capabilities
- consul integration
- docker driver
- add maximal job lifetime
- add job.OnError {reschedule, script callback, nothing} (Exec on server)
- add job.OnSuccess {script callback, nothing} (Exec on server)
- add job.OnUpdate {script callback, nothing} (Exec on server)
- add TLS
- -dev mode
- cmd line args
- docs
- remove deletion of database on startup :D :D
