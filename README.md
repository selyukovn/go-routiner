# Goroutiner

## TL;DR

Fluent, zero‑boilerplate goroutine launcher with middleware support.

Requires Go (1.18) or later.

---

## Description

This package allows wrapping goroutines with middleware
and eliminates boilerplate code for typical execution strategies.

### Middleware

There are 3 middleware sets:

- `global` -- wraps any goroutine launched by a goroutiner instance.
- `batch` -- wraps any goroutine launched in a `Batch` instance.
- `individual` -- wraps only currently adding goroutine.

```
var globalMws := []goroutiner.Middleware
var batchMws := []goroutiner.Middleware
var mwsForGo1 := []goroutiner.Middleware
var mwsForGo2 := []goroutiner.Middleware

_ = goroutiner.
    New(globalMws...).
    Batch(ctx, batchMws...).
    Add(go1, mwsForGo1...).
    Add(go2, mwsForGo2...).
    Add(go3, mwsForGo2...).
    Wait()
    
// Result middlewares:
- goroutine 1: globalMws + batchMws + mwsForGo1
- goroutine 2: globalMws + batchMws + mwsForGo2
- goroutine 3: globalMws + batchMws
```

Package provides typically needed middleware for handling panics:

- `MwPanicToError` -- converts panics to errors, can be used to prevent the app from crashing
- `MwPanicRelay` -- allows custom handling of panics before they propagate further (e.g., for logging purposes)

### Execution strategies

Most typical execution strategies are:

- via `sync.WaitGroup` -- when need to wait for all goroutines
- via [`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup) --
  when need to wait for the first fail only
- via simple `go func() { ... }()` -- when no need to wait

Goroutiner provides a wrapper method for each of these strategies to reduce boilerplate code:

- `Wait()`
- `CancelOnError()`
- `Async()`

Also, there are additional methods for more specific cases:

- `SingleAsync()` -- for cases, when only one goroutine needs to be launched in async mode
- `AsyncBs()` -- for cases, when need to execute `Async` with custom result channel buffer size

---

## Examples

### Case 1: Wait() + Global Middleware

- Wait for 2 goroutines and collect errors.
- If panic occurs in any goroutine, log it properly before the app fails.

<table style="width: 100%">
    <thead>
        <th>Manual code</th>
        <th>This package</th>
    </thead>
    <tbody>
        <tr>
            <td style="vertical-align: top;">
<pre>
// Some initializations -- once for all time.
// --------------------------------------------

// ...
</pre>
            </td>
            <td style="vertical-align: top;">
<pre>
// Some initializations -- once for all time.
// --------------------------------------------

// ...

logPanicMw := goroutiner.MwPanicRelay(func(p any, ds []byte, ctx context.Context) any {
    // ... log properly ...
    return p
})

grt := goroutiner.New(logPanicMw)
</pre>
            </td>
        </tr>
        <tr>
            <td style="vertical-align: top;">
<pre>
// Executing -- each time such case is needed.
// --------------------------------------------

var err1 error
var err2 error

wg := new(sync.WaitGroup)
wg.Add(2)

go func() {
    defer wg.Done()
    defer func() {
        if p := recover(); p != nil {
            ds := debug.Stack()
            // ... log properly ...
            panic(p)
        }
    }()
    // ... do something ...
    err1 = errors.New("error from goroutine #1")
}()

go func() {
    defer wg.Done()
    defer func() {
        if p := recover(); p != nil {
            ds := debug.Stack()
            // ... log properly ...
            panic(p)
        }
    }()
    // ... do something else ...
    err2 = errors.New("error from goroutine #2")
}()

wg.Wait()
</pre>
            </td>
            <td style="vertical-align: top;">
<pre>
// Executing -- each time such case is needed.
// --------------------------------------------

errs := grt.Batch(ctx).
    Add(func(ctx context.Context) error {
        // ... do something ...
        return errors.New("error from goroutine #1")
    }).
    Add(func(ctx context.Context) error {
        // ... do something else ...
        return errors.New("error from goroutine #2")
    }).
    Wait()
</pre>
            </td>
        </tr>
    </tbody>
</table>

### Case 2: CancelOnError() + Session Middleware

- Wait for 2 requests (e.g. to combine results later)
- If any request fails with an error or panics, return error and cancel other requests.

<table style="width: 100%">
    <thead>
        <th>Manual code</th>
        <th>This package</th>
    </thead>
    <tbody>
        <tr>
            <td style="vertical-align: top;">
<pre>
// Some initializations -- once for all time.
// --------------------------------------------

// ...
</pre>
            </td>
            <td style="vertical-align: top;">
<pre>
// Some initializations -- once for all time.
// --------------------------------------------

// ...
</pre>
            </td>
        </tr>
        <tr>
            <td style="vertical-align: top;">
<pre>
// Executing -- each time such case is needed.
// --------------------------------------------

var res1 any
var res2 any

eg, erCtx := errgroup.WithContext(ctx)

er.Go(func() (rErr error) {
    defer func() {
        if p := recover(); p != nil {
            ds := debug.Stack()
            // ... log properly ...
            rErr = fmt.Errorf("panic: %#v; stack: %s", p, ds)
        }
    }()
    res, err := ... do request #1 ...
    res1 = res
    rErr = err
})

er.Go(func() (rErr error) {
    defer func() {
        if p := recover(); p != nil {
            ds := debug.Stack()
            // ... log properly ...
            rErr = fmt.Errorf("panic: %#v; stack: %s", p, ds)
        }
    }()
    res, err := ... do request #2 ...
    res1 = res
    rErr = err
})

err := eg.Wait()
</pre>
            </td>
            <td style="vertical-align: top;">
<pre>
// Executing -- each time such case is needed.
// --------------------------------------------

var res1 any
var res2 any

err := goroutiner.New().
    Batch(ctx, goroutiner.MwPanicToError(func(p any, ds []byte, ctx context.Context) error {
        return fmt.Errorf("panic: %#v; stack: %s", p, ds)
    })).
    Add(func(ctx context.Context) error {
        res, err := ... do request #1 ...
        res1 = res
        return err
    }).
    Add(func(ctx context.Context) error {
        res, err := ... do request #2 ...
        res2 = res
        return err
    }).
    CancelOnError()
</pre>
            </td>
        </tr>
    </tbody>
</table>

### Case 3: Async + Individual Middleware

- Send email-message asynchronously
- Notify client code once completed
- Do not fail the app even if sender panics

<table style="width: 100%">
    <thead>
        <th>Manual code</th>
        <th>This package</th>
    </thead>
    <tbody>
        <tr>
            <td style="vertical-align: top;">
<pre>
// Some initializations -- once for all time.
// --------------------------------------------

// ...
</pre>
            </td>
            <td style="vertical-align: top;">
<pre>
// Some initializations -- once for all time.
// --------------------------------------------

// ...
</pre>
            </td>
        </tr>
        <tr>
            <td style="vertical-align: top;">
<pre>
// Executing -- each time such case is needed.
// --------------------------------------------

errCh := make(chan error, 1)
go func() {
    defer close(errCh)
    defer func() {
        if p := recover(); p != nil {
            ds := string(debug.Stack())
            errCh <- fmt.Errorf("panic: %#v; stack: %s", p, ds)
        }
    }()
    err := ... send email ...
    errCh <- err
}()
</pre>
            </td>
            <td style="vertical-align: top;">
<pre>
// Executing -- each time such case is needed.
// --------------------------------------------

errCh := goroutiner.New().SingleAsync(ctx, func(ctx context.Context) error {
        return ... send email ...
    }, goroutiner.MwPanicToError(func(p any, ds []byte, ctx context.Context) error {
        return fmt.Errorf("panic: %#v; stack: %s", p, ds)
    }))
</pre>
            </td>
        </tr>
    </tbody>
</table>
