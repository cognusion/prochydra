

# head
`import "github.com/cognusion/prochydra/head"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>



## <a name="pkg-index">Index</a>
* [func ReadLogger(reader io.Reader, writer *log.Logger, errorChan chan&lt;- error)](#ReadLogger)
* [type Head](#Head)
  * [func BashDashC(command string, errorChan chan error) *Head](#BashDashC)
  * [func New(command string, args []string, errorChan chan error) *Head](#New)
  * [func (r *Head) Autorestart(doit bool)](#Head.Autorestart)
  * [func (r *Head) Clone() *Head](#Head.Clone)
  * [func (r *Head) Errors() uint64](#Head.Errors)
  * [func (r *Head) Restarts() uint64](#Head.Restarts)
  * [func (r *Head) RestartsPerMinute() uint64](#Head.RestartsPerMinute)
  * [func (r *Head) Run() string](#Head.Run)
  * [func (r *Head) SetChildEnv(env []string)](#Head.SetChildEnv)
  * [func (r *Head) SetMgInterval(interval time.Duration)](#Head.SetMgInterval)
  * [func (r *Head) Stop()](#Head.Stop)
  * [func (r *Head) String() string](#Head.String)
  * [func (r *Head) Wait()](#Head.Wait)
  * [func (r *Head) Write(p []byte) (n int, err error)](#Head.Write)
* [type Sbuffer](#Sbuffer)
  * [func (s *Sbuffer) String() string](#Sbuffer.String)
  * [func (s *Sbuffer) Write(p []byte) (n int, err error)](#Sbuffer.Write)


#### <a name="pkg-files">Package files</a>
[common.go](https://github.com/cognusion/prochydra/tree/master/head/common.go) [head.go](https://github.com/cognusion/prochydra/tree/master/head/head.go)





## <a name="ReadLogger">func</a> [ReadLogger](https://github.com/cognusion/prochydra/tree/master/head/common.go?s=156:233#L12)
``` go
func ReadLogger(reader io.Reader, writer *log.Logger, errorChan chan<- error)
```
ReadLogger continuously reads from an io.Reader, and blurts to the specified log.Logger




## <a name="Head">type</a> [Head](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=442:2298#L26)
``` go
type Head struct {
    // ID is a generated ID based on the sequence
    ID string
    // RestartDelay specified the duration to wait between restarts
    RestartDelay time.Duration
    // MaxPSS specifies the maximum PSS size a process may have before being killed
    MaxPSS int64
    // DebugOut is a logger for debug information
    DebugOut *log.Logger
    // ErrOut is a logger for errors, generally, emitted by a Head
    ErrOut *log.Logger
    // StdErr is a logger for StdErr coming from a process
    StdErr *log.Logger
    // StdOut is a logger for StdOut coming from a process
    StdOut *log.Logger
    // UID is the uid to run as (leave unset for current user)
    UID uint32
    // GID is the gid to run as (leave unset for current group)
    GID uint32
    // Seq is a pointer to an initialized sequencer
    Seq *sequence.Seq
    // Values is a map for implementors to store key-value pairs. Never consulted by the Head library.
    Values sync.Map
    // Timeout is a duration after which the process running is stopped, subject to  Autorestart
    Timeout time.Duration
    // StdInNoNL is a boolean to describe if a NewLine should *not* be appended to lines written to StdIn.
    // This is advisory-only, and respected by hydra but not necessarily others.
    StdInNoNL bool
    // StdInShellEscapeInput is a boolean to describe if strings send to StdIn should be shell-escaped.
    // This is advisory-only, and respected by hydra but not necessarily others.
    StdInShellEscapeInput bool
    // contains filtered or unexported fields
}

```
Head is a struct to contain a process a run. You can Run() the same Head multiple times if
you need clones.







### <a name="BashDashC">func</a> [BashDashC](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=2400:2458#L77)
``` go
func BashDashC(command string, errorChan chan error) *Head
```
BashDashC creates a head that handles the command in its entirety running as a "bash -c command"


### <a name="New">func</a> [New](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=2715:2782#L84)
``` go
func New(command string, args []string, errorChan chan error) *Head
```
New returns a Head struct, ready to Run() the command with the arguments
specified. It would be wise to ensure the errorChan reader is ready before calling
Run() to prevent goro plaque.





### <a name="Head.Autorestart">func</a> (\*Head) [Autorestart](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=4257:4294#L141)
``` go
func (r *Head) Autorestart(doit bool)
```
Autorestart sets whether or not we will automatically restart Heads that "complete"




### <a name="Head.Clone">func</a> (\*Head) [Clone](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=3427:3455#L108)
``` go
func (r *Head) Clone() *Head
```
Clone returns a new Head intialized the same as the current




### <a name="Head.Errors">func</a> (\*Head) [Errors](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=9334:9364#L336)
``` go
func (r *Head) Errors() uint64
```
Errors returns the current number of errors sent to the error chan




### <a name="Head.Restarts">func</a> (\*Head) [Restarts](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=9481:9513#L341)
``` go
func (r *Head) Restarts() uint64
```
Restarts returns the current number of restarts for this Head instance




### <a name="Head.RestartsPerMinute">func</a> (\*Head) [RestartsPerMinute](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=9655:9696#L347)
``` go
func (r *Head) RestartsPerMinute() uint64
```
RestartsPerMinute returns the number of restarts for this Head instance in
the last minute




### <a name="Head.Run">func</a> (\*Head) [Run](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=4897:4924#L159)
``` go
func (r *Head) Run() string
```
Run executes a subprocess of the command and arguments specified, restarting
it if applicable. The returned channel returns the name of the once it is
running, or closes it without a value if it will not run.




### <a name="Head.SetChildEnv">func</a> (\*Head) [SetChildEnv](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=4106:4146#L136)
``` go
func (r *Head) SetChildEnv(env []string)
```
SetChildEnv takes a list of key=value strings to pass to all spawned processes




### <a name="Head.SetMgInterval">func</a> (\*Head) [SetMgInterval](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=4437:4489#L147)
``` go
func (r *Head) SetMgInterval(interval time.Duration)
```
SetMgInterval sets the interval at which the memory usage is checked, for use with MaxPSS.
Default 30s.




### <a name="Head.Stop">func</a> (\*Head) [Stop](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=8995:9016#L327)
``` go
func (r *Head) Stop()
```
Stop signals all of the running processes to die. May generate error
output thereafter.




### <a name="Head.String">func</a> (\*Head) [String](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=4583:4613#L152)
``` go
func (r *Head) String() string
```
String returns the original command and arguments in a line




### <a name="Head.Wait">func</a> (\*Head) [Wait](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=9793:9814#L352)
``` go
func (r *Head) Wait()
```
Wait blocks until the Run() process has completed




### <a name="Head.Write">func</a> (\*Head) [Write](https://github.com/cognusion/prochydra/tree/master/head/head.go?s=3896:3945#L129)
``` go
func (r *Head) Write(p []byte) (n int, err error)
```
Write implements io.Writer to ease writing to StdIn




## <a name="Sbuffer">type</a> [Sbuffer](https://github.com/cognusion/prochydra/tree/master/head/common.go?s=731:794#L42)
``` go
type Sbuffer struct {
    // contains filtered or unexported fields
}

```
Sbuffer is a goro-safe bytes.Buffer










### <a name="Sbuffer.String">func</a> (\*Sbuffer) [String](https://github.com/cognusion/prochydra/tree/master/head/common.go?s=1135:1168#L56)
``` go
func (s *Sbuffer) String() string
```
String returns the contents of the unread portion of the buffer as a string.




### <a name="Sbuffer.Write">func</a> (\*Sbuffer) [Write](https://github.com/cognusion/prochydra/tree/master/head/common.go?s=931:983#L49)
``` go
func (s *Sbuffer) Write(p []byte) (n int, err error)
```
Write appends the contents of p to the buffer, growing the buffer as needed. It returns
the number of bytes written or an error.








- - -
Generated by [godoc2md](http://github.com/cognusion/godoc2md)
