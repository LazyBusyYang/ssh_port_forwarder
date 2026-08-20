package ssh_manager

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ssh-port-forwarder/internal/config"
	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/pkg/crypto"
)

type trackingListener struct {
	closeOnce     sync.Once
	startOnce     sync.Once
	closedCh      chan struct{}
	acceptStarted chan struct{}
	acceptCalls   atomic.Int32
}

func newTrackingListener() *trackingListener {
	return &trackingListener{
		closedCh:      make(chan struct{}),
		acceptStarted: make(chan struct{}),
	}
}

func (l *trackingListener) Accept() (net.Conn, error) {
	l.acceptCalls.Add(1)
	l.startOnce.Do(func() { close(l.acceptStarted) })
	<-l.closedCh
	return nil, net.ErrClosed
}

type delayedListener struct {
	startOnce     sync.Once
	releaseOnce   sync.Once
	acceptStarted chan struct{}
	releaseCh     chan struct{}
}

func newDelayedListener() *delayedListener {
	return &delayedListener{
		acceptStarted: make(chan struct{}),
		releaseCh:     make(chan struct{}),
	}
}

func (l *delayedListener) Accept() (net.Conn, error) {
	l.startOnce.Do(func() { close(l.acceptStarted) })
	<-l.releaseCh
	return nil, net.ErrClosed
}
func (l *delayedListener) Addr() net.Addr { return nil }
func (l *delayedListener) Close() error   { return nil }
func (l *delayedListener) release() {
	l.releaseOnce.Do(func() { close(l.releaseCh) })
}

func waitForSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}
func (l *trackingListener) Addr() net.Addr { return nil }
func (l *trackingListener) Close() error {
	l.closeOnce.Do(func() { close(l.closedCh) })
	return nil
}
func (l *trackingListener) isClosed() bool {
	select {
	case <-l.closedCh:
		return true
	default:
		return false
	}
}

func testClientWithForward(t *testing.T, hostID, ruleID uint64) (*SSHClient, *trackingListener) {
	t.Helper()

	listener := newTrackingListener()

	client := NewSSHClient(&model.SSHHost{ID: hostID}, nil)
	client.state = ConnStateConnected
	client.forwards[ruleID] = &ForwardEntry{
		RuleID:   ruleID,
		listener: listener,
		stopCh:   make(chan struct{}),
		active:   true,
	}
	return client, listener
}

func TestStopForwardRuleEverywhereIgnoresDatabaseOwner(t *testing.T) {
	manager := &Manager{clients: make(map[uint64]*SSHClient)}
	client, listener := testClientWithForward(t, 2, 27)
	manager.clients[2] = client

	stopped, err := manager.StopForwardRuleEverywhere(27)
	if err != nil {
		t.Fatalf("stop forward everywhere: %v", err)
	}
	if stopped != 1 {
		t.Fatalf("stopped = %d, want 1", stopped)
	}
	if client.GetForward(27) != nil {
		t.Fatal("forward entry was not removed")
	}
	if !listener.isClosed() {
		t.Fatal("listener remained open")
	}
}

func TestDisconnectClosesForwardsAndIsIdempotent(t *testing.T) {
	client, listener := testClientWithForward(t, 4, 27)
	client.state = ConnStateFailed

	if err := client.Disconnect(); err != nil {
		t.Fatalf("first disconnect: %v", err)
	}
	if err := client.Disconnect(); err != nil {
		t.Fatalf("second disconnect: %v", err)
	}
	if client.GetForward(27) != nil {
		t.Fatal("forward entry was not removed")
	}
	if !listener.isClosed() {
		t.Fatal("listener remained open")
	}
}

func TestConnectHostCleansStaleClientBeforeReplacement(t *testing.T) {
	const encryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

	ciphertext, nonce, err := crypto.Encrypt("unused", encryptionKey)
	if err != nil {
		t.Fatalf("encrypt fixture: %v", err)
	}

	manager := NewManager(nil, nil, config.EncryptionConfig{Key: encryptionKey})
	staleClient, listener := testClientWithForward(t, 4, 27)
	staleClient.state = ConnStateFailed
	manager.clients[4] = staleClient

	err = manager.ConnectHost(&model.SSHHost{
		ID:         4,
		AuthMethod: "unsupported",
		AuthData:   ciphertext,
		AuthNonce:  nonce,
	})
	if err == nil {
		t.Fatal("ConnectHost unexpectedly succeeded")
	}
	if !listener.isClosed() {
		t.Fatal("stale client listener remained open")
	}
	if manager.GetClient(4) != nil {
		t.Fatal("stale client remained registered")
	}
}

func TestReconnectGuardAllowsFirstLoopOnly(t *testing.T) {
	client := NewSSHClient(&model.SSHHost{ID: 4}, nil)
	client.state = ConnStateConnected

	if !client.beginReconnect() {
		t.Fatal("first reconnect attempt was rejected")
	}
	if client.beginReconnect() {
		t.Fatal("duplicate reconnect attempt was accepted")
	}
	client.endReconnect()
	if !client.beginReconnect() {
		t.Fatal("reconnect guard was not released")
	}
	client.endReconnect()
}

func TestStopDuringReconnectPreventsRuleRestore(t *testing.T) {
	client, oldListener := testClientWithForward(t, 4, 27)
	manager := &Manager{clients: map[uint64]*SSHClient{4: client}}
	restoreCalls := 0
	client.listenerFactory = func(string) (net.Listener, error) {
		restoreCalls++
		return newTrackingListener(), nil
	}

	client.SuspendAllForwards()
	if !oldListener.isClosed() {
		t.Fatal("listener was not closed during reconnect suspension")
	}
	if client.GetForward(27) == nil {
		t.Fatal("suspended rule was removed before restart/delete could cancel it")
	}

	stopped, err := manager.StopForwardRuleEverywhere(27)
	if err != nil {
		t.Fatalf("stop rule during reconnect: %v", err)
	}
	if stopped != 1 {
		t.Fatalf("stopped = %d, want 1", stopped)
	}
	client.RestoreSuspendedForwards()

	if restoreCalls != 0 {
		t.Fatalf("deleted rule listener was restored %d times", restoreCalls)
	}
	if client.GetForward(27) != nil {
		t.Fatal("deleted rule was restored into the runtime map")
	}
}

func TestReconnectRestoresRuleStillRegistered(t *testing.T) {
	client, oldListener := testClientWithForward(t, 4, 27)
	restoredListener := newTrackingListener()
	restoreCalls := 0
	client.listenerFactory = func(addr string) (net.Listener, error) {
		restoreCalls++
		if addr != "0.0.0.0:30003" {
			t.Fatalf("restore address = %q", addr)
		}
		return restoredListener, nil
	}
	entry := client.GetForward(27)
	entry.LocalAddr = "0.0.0.0:30003"
	entry.RemoteAddr = "10.119.18.110:28080"

	client.SuspendAllForwards()
	client.RestoreSuspendedForwards()

	if !oldListener.isClosed() {
		t.Fatal("old listener was not closed")
	}
	if restoreCalls != 1 {
		t.Fatalf("restore calls = %d, want 1", restoreCalls)
	}
	restored := client.GetForward(27)
	if restored == nil || !restored.active || restored.listener != restoredListener {
		t.Fatal("registered rule was not restored")
	}

	if err := client.StopForward(27); err != nil {
		t.Fatalf("cleanup restored rule: %v", err)
	}
}

func TestOldAcceptLoopCannotConsumeRestoredGeneration(t *testing.T) {
	client := NewSSHClient(&model.SSHHost{ID: 4}, nil)
	oldListener := newDelayedListener()
	oldStopCh := make(chan struct{})
	entry := &ForwardEntry{
		RuleID:     27,
		LocalAddr:  "0.0.0.0:30003",
		RemoteAddr: "10.119.18.110:28080",
		listener:   oldListener,
		stopCh:     oldStopCh,
		active:     true,
	}
	client.forwards[27] = entry

	oldLoopDone := make(chan struct{})
	go func() {
		client.acceptConnections(oldListener, oldStopCh, entry.LocalAddr, entry.RemoteAddr)
		close(oldLoopDone)
	}()
	waitForSignal(t, oldListener.acceptStarted, "old listener accept")

	newListener := newTrackingListener()
	client.listenerFactory = func(string) (net.Listener, error) {
		return newListener, nil
	}
	client.SuspendAllForwards()
	client.RestoreSuspendedForwards()
	waitForSignal(t, newListener.acceptStarted, "new listener accept")

	oldListener.release()
	waitForSignal(t, oldLoopDone, "old accept loop exit")
	if calls := newListener.acceptCalls.Load(); calls != 1 {
		t.Fatalf("new listener accepted by %d loops, want 1", calls)
	}

	if err := client.StopForward(27); err != nil {
		t.Fatalf("cleanup restored rule: %v", err)
	}
}
