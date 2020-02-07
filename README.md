# taylor
distributed worker queue written in golang

## Dependencies

- uuid
- gin
- ql

## To-Do

- cancel job REST and tcp to agent who executes
- consul integration
- docker driver
- add maximal job lifetime (to forecome infinite loop in job exec)
- add possibility to distribute job scripts to agents
- add job.OnError {reschedule, script callback, nothing} (Exec on server)
- add job.OnSuccess {script callback, nothing} (Exec on server)
- add job.OnUpdate {script callback, nothing} (Exec on server)
- add TLS
- cmd line args
- docs
