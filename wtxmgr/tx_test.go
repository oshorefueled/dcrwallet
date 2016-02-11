// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2015 The Decred developers
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package wtxmgr_test

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/decred/dcrd/chaincfg"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrutil"
	"github.com/decred/dcrwallet/walletdb"
	_ "github.com/decred/dcrwallet/walletdb/bdb"
	"github.com/decred/dcrwallet/wtxmgr"
	. "github.com/decred/dcrwallet/wtxmgr"
)

// Received transaction output for mainnet outpoint
// 61d3696de4c888730cbe06b0ad8ecb6d72d6108e893895aa9bc067bd7eba3fad:0
var (
	TstRecvSerializedTx, _          = hex.DecodeString("01000000020000000000000000000000000000000000000000000000000000000000000000ffffffff0000ffffffff04a2b971f28a2068ca54c6ba47507be802b3204264c67a8d8ce1fa7b34980908000000000100ffffffff0300000000000000002a6a28147ecac1bb4fe2db0246bebf91ea469c7ae53bd764c8f97edb7e9d637d8100035c0c0000000000000000000000000000046a02ffffb8101d1e000000001abb76a91414ab317ede91b40545f2e3b32678f9b6cc21b69e88ac000000000204deadbeef6a47304402200d590d4da7cea182fe9f2f812f1f42ac2ab8217018a5f423774f8f9c9f665c8f02200128fc6222d7daa44f5492aae5f60345e4a78935fafa6c73d2e531f4a297405f012103ca49eba5fb8d6c881e8a7385cc3c027971257d938de9e0ea6fa8426268703501")
	TstRecvTx, _                    = dcrutil.NewTxFromBytes(TstRecvSerializedTx)
	TstRecvTxSpendingTxBlockHash, _ = chainhash.NewHashFromStr("60be28d1fa9721a22a4346245fa9694bde31ea2c8f29e49fdcd5d0a00e59bf4e")
	TstRecvAmt                      = int64(100)
	TstRecvTxBlockDetails           = &BlockMeta{
		Block: Block{Hash: *TstRecvTxSpendingTxBlockHash, Height: 762},
		Time:  time.Unix(1387737310, 0),
	}

	TstRecvCurrentHeight = int32(284498) // mainnet blockchain height at time of writing
	TstRecvTxOutConfirms = 8074          // hardcoded number of confirmations given the above block height

	TstSpendingSerializedTx, _ = hex.DecodeString("01000000042238516e1d48beb5b0c90dfcbaff751088237efc09c13cf66178969fe0d880cb0a000000016a47304402205ae23e9fb4ab4b4b23a7c762df8541b3313197d2f408337da91973ef97eac66202204734fbabed53405febfe64f4cebad29b8434f4f69c56879f1a703d997ea395a90121036b392a1272ca0fbe04de90e48ef872ef470839887842ad5520fd3d79b42ed223ffffffff308c17f850f1b18034da3446a8dc0d69406ed1f37683b6f43071f0070de27d6003000000016b483045022100eac9a62e4499dd80b20c16c6a65c9ecab1e1e4bf00c85f8fb1691cd5c8e9628d02202b79b30b6fc4cbee8cca548093040b5bb05c15a79f4d0664a2ef0f909150f1cf0121031f9913f6db4884a463e77c050fae18626376c0cf8e20232c9de40aeebcd7c457ffffffff3241e3b556db727f6003613fbbb38d7c409f6ac0971c200d6052cc7dde68dd5904000000016a47304402207d08ebb2f70856624e5ac1054c3c6ae289c280e6ed47877f05395f6f7a1ef7880220264b011099af48d548b12ac5e69f6a2abadded6ad68dbbe3f716bec9c806edab0121023ccb059324229c1149701380468b4b656f043ea75bf785b9ed83a0700650a08dffffffff2c4e4360883e113cde3c6857d71e65e822289ef13e4b2a3229b42a65f9b7b93b03000000006a473044022020d50f153f5a0dadbbaedd2e76a7f12fb4c06c09e57fe9927187739da5695a350220648cc08901fa238007525b4cfd0e0014b5e8f5546200bf56797456dd6d95f73c0121032501f9806b6e00949f91105b4363b10507352ca42ecc4260842259bb45f3fcd0ffffffff0200e40b54020000001976a914de6d4ab876fd17638a0f651dda6b9e5f84637acb88ac34849525000000001976a9140bf5130cf9c0e8a10a7da6f6d90961335783fd7b88ac00000000")
	TstSpendingTx, _           = dcrutil.NewTxFromBytes(TstSpendingSerializedTx)
	TstSpendingTxBlockHeight   = int32(279143)
	TstSignedTxBlockHash, _    = chainhash.NewHashFromStr("00000000000000017188b968a371bab95aa43522665353b646e41865abae02a4")
	TstSignedTxBlockDetails    = &BlockMeta{
		Block: Block{Hash: *TstSignedTxBlockHash, Height: TstSpendingTxBlockHeight},
		Time:  time.Unix(1389114091, 0),
	}
)

func testDB() (walletdb.DB, func(), error) {
	tmpDir, err := ioutil.TempDir("", "wtxmgr_test")
	if err != nil {
		return nil, func() {}, err
	}
	db, err := walletdb.Create("bdb", filepath.Join(tmpDir, "db"))
	return db, func() { os.RemoveAll(tmpDir) }, err
}

func testStore() (*Store, func(), error) {
	tmpDir, err := ioutil.TempDir("", "wtxmgr_test")
	if err != nil {
		return nil, func() {}, err
	}
	db, err := walletdb.Create("bdb", filepath.Join(tmpDir, "db"))
	if err != nil {
		teardown := func() {
			os.RemoveAll(tmpDir)
		}
		return nil, teardown, err
	}
	teardown := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	ns, err := db.Namespace([]byte("txstore"))
	if err != nil {
		return nil, teardown, err
	}
	s, err := Create(ns, &chaincfg.TestNetParams)
	return s, teardown, err
}

func serializeTx(tx *dcrutil.Tx) []byte {
	var buf bytes.Buffer
	err := tx.MsgTx().Serialize(&buf)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestInsertsCreditsDebitsRollbacks(t *testing.T) {
	t.Parallel()

	// Create a double spend of the received blockchain transaction.
	dupRecvTx, err := dcrutil.NewTxFromBytesLegacy(TstRecvSerializedTx)
	if err != nil {
		t.Errorf("failed to deserialize test transaction: %v", err.Error())
		return
	}

	// Switch txout amount to 1 DCR.  Transaction store doesn't
	// validate txs, so this is fine for testing a double spend
	// removal.

	TstDupRecvAmount := int64(1e8)
	newDupMsgTx := dupRecvTx.MsgTx()
	newDupMsgTx.TxOut[0].Value = TstDupRecvAmount
	TstDoubleSpendTx := dcrutil.NewTx(newDupMsgTx)
	TstDoubleSpendSerializedTx := serializeTx(TstDoubleSpendTx)

	// Create a "signed" (with invalid sigs) tx that spends output 0 of
	// the double spend.
	spendingTx := wire.NewMsgTx()
	spendingTxIn := wire.NewTxIn(wire.NewOutPoint(TstDoubleSpendTx.Sha(), 0, dcrutil.TxTreeRegular), []byte{0, 1, 2, 3, 4})
	spendingTx.AddTxIn(spendingTxIn)
	spendingTxOut1 := wire.NewTxOut(1e7, []byte{5, 6, 7, 8, 9})
	spendingTxOut2 := wire.NewTxOut(9e7, []byte{10, 11, 12, 13, 14})
	spendingTx.AddTxOut(spendingTxOut1)
	spendingTx.AddTxOut(spendingTxOut2)
	TstSpendingTx := dcrutil.NewTx(spendingTx)
	TstSpendingSerializedTx := serializeTx(TstSpendingTx)
	var _ = TstSpendingTx

	tests := []struct {
		name     string
		f        func(*Store) (*Store, error)
		bal, unc dcrutil.Amount
		unspents map[wire.OutPoint]struct{}
		unmined  map[chainhash.Hash]struct{}
	}{
		{
			name: "new store",
			f: func(s *Store) (*Store, error) {
				return s, nil
			},
			bal:      0,
			unc:      0,
			unspents: map[wire.OutPoint]struct{}{},
			unmined:  map[chainhash.Hash]struct{}{},
		},
		{
			name: "txout insert",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, nil)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(rec, nil, 0, false)
				return s, err
			},
			bal: 0,
			unc: dcrutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstRecvTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstRecvTx.Sha(): {},
			},
		},
		{
			name: "insert duplicate unconfirmed",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, nil)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(rec, nil, 0, false)
				return s, err
			},
			bal: 0,
			unc: dcrutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstRecvTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstRecvTx.Sha(): {},
			},
		},
		{
			name: "confirmed txout insert",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: dcrutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstRecvTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "insert duplicate confirmed",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: dcrutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstRecvTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "rollback confirmed credit",
			f: func(s *Store) (*Store, error) {
				err := s.Rollback(TstRecvTxBlockDetails.Height)
				return s, err
			},
			bal: 0,
			unc: dcrutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstRecvTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstRecvTx.Sha(): {},
			},
		},
		{
			name: "insert confirmed double spend",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstDoubleSpendSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: dcrutil.Amount(TstDoubleSpendTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstDoubleSpendTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "insert unconfirmed debit",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, nil)
				return s, err
			},
			bal:      0,
			unc:      0,
			unspents: map[wire.OutPoint]struct{}{},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Sha(): {},
			},
		},
		{
			name: "insert unconfirmed debit again",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstDoubleSpendSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, TstRecvTxBlockDetails)
				return s, err
			},
			bal:      0,
			unc:      0,
			unspents: map[wire.OutPoint]struct{}{},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Sha(): {},
			},
		},
		{
			name: "insert change (index 0)",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, nil)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(rec, nil, 0, true)
				return s, err
			},
			bal: 0,
			unc: dcrutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstSpendingTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Sha(): {},
			},
		},
		{
			name: "insert output back to this own wallet (index 1)",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, nil)
				if err != nil {
					return nil, err
				}
				err = s.AddCredit(rec, nil, 1, true)
				return s, err
			},
			bal: 0,
			unc: dcrutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstSpendingTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
				wire.OutPoint{*TstSpendingTx.Sha(), 1, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Sha(): {},
			},
		},
		{
			name: "confirm signed tx",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, TstSignedTxBlockDetails)
				return s, err
			},
			bal: dcrutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstSpendingTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
				wire.OutPoint{*TstSpendingTx.Sha(), 1, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "rollback after spending tx",
			f: func(s *Store) (*Store, error) {
				err := s.Rollback(TstSignedTxBlockDetails.Height + 1)
				return s, err
			},
			bal: dcrutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstSpendingTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
				wire.OutPoint{*TstSpendingTx.Sha(), 1, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "rollback spending tx block",
			f: func(s *Store) (*Store, error) {
				err := s.Rollback(TstSignedTxBlockDetails.Height)
				return s, err
			},
			bal: 0,
			unc: dcrutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{*TstSpendingTx.Sha(), 0, dcrutil.TxTreeRegular}: {},
				wire.OutPoint{*TstSpendingTx.Sha(), 1, dcrutil.TxTreeRegular}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Sha(): {},
			},
		},
		{
			name: "rollback double spend tx block",
			f: func(s *Store) (*Store, error) {
				err := s.Rollback(TstRecvTxBlockDetails.Height)
				return s, err
			},
			bal: 0,
			unc: dcrutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unspents: map[wire.OutPoint]struct{}{
				*wire.NewOutPoint(TstSpendingTx.Sha(), 0, dcrutil.TxTreeRegular): {},
				*wire.NewOutPoint(TstSpendingTx.Sha(), 1, dcrutil.TxTreeRegular): {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstDoubleSpendTx.Sha(): {},
				*TstSpendingTx.Sha():    {},
			},
		},
		{
			name: "insert original recv txout",
			f: func(s *Store) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}
				err = s.AddCredit(rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: dcrutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				*wire.NewOutPoint(TstRecvTx.Sha(), 0, dcrutil.TxTreeRegular): {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
	}

	s, teardown, err := testStore()
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		tmpStore, err := test.f(s)
		if err != nil {
			t.Fatalf("%s: got error: %v", test.name, err)
		}
		s = tmpStore
		bal, err := s.Balance(1, TstRecvCurrentHeight, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("%s: Confirmed Balance failed: %v", test.name, err)
		}
		if bal != test.bal {
			t.Fatalf("%s: balance mismatch: expected: %d, got: %d", test.name, test.bal, bal)
		}
		unc, err := s.Balance(0, TstRecvCurrentHeight, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("%s: Unconfirmed Balance failed: %v", test.name, err)
		}
		unc -= bal
		if unc != test.unc {
			t.Fatalf("%s: unconfirmed balance mismatch: expected %d, got %d", test.name, test.unc, unc)
		}

		// Check that unspent outputs match expected.
		unspent, err := s.UnspentOutputs()
		if err != nil {
			t.Fatalf("%s: failed to fetch unspent outputs: %v", test.name, err)
		}
		for _, cred := range unspent {
			if _, ok := test.unspents[cred.OutPoint]; !ok {
				t.Errorf("%s: unexpected unspent output: %v", test.name, cred.OutPoint)
			}
			delete(test.unspents, cred.OutPoint)
		}
		if len(test.unspents) != 0 {
			t.Fatalf("%s: missing expected unspent output(s)", test.name)
		}

		// Check that unmined txs match expected.
		unmined, err := s.UnminedTxs()
		if err != nil {
			t.Fatalf("%s: cannot load unmined transactions: %v", test.name, err)
		}
		for _, tx := range unmined {
			txHash := tx.TxSha()
			if _, ok := test.unmined[txHash]; !ok {
				t.Fatalf("%s: unexpected unmined tx: %v", test.name, txHash)
			}
			delete(test.unmined, txHash)
		}
		if len(test.unmined) != 0 {
			t.Fatalf("%s: missing expected unmined tx(s)", test.name)
		}

	}
}

func TestFindingSpentCredits(t *testing.T) {
	t.Parallel()

	s, teardown, err := testStore()
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	// Insert transaction and credit which will be spent.
	recvRec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = s.InsertTx(recvRec, TstRecvTxBlockDetails)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(recvRec, TstRecvTxBlockDetails, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	// Insert confirmed transaction which spends the above credit.
	spendingRec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = s.InsertTx(spendingRec, TstSignedTxBlockDetails)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(spendingRec, TstSignedTxBlockDetails, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	bal, err := s.Balance(1, TstSignedTxBlockDetails.Height, wtxmgr.BFBalanceSpendable)
	if err != nil {
		t.Fatal(err)
	}
	expectedBal := dcrutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value)
	if bal != expectedBal {
		t.Fatalf("bad balance: %v != %v", bal, expectedBal)
	}
	unspents, err := s.UnspentOutputs()
	if err != nil {
		t.Fatal(err)
	}
	op := wire.NewOutPoint(TstSpendingTx.Sha(), 0, dcrutil.TxTreeRegular)
	if unspents[0].OutPoint != *op {
		t.Fatal("unspent outpoint doesn't match expected")
	}
	if len(unspents) > 1 {
		t.Fatal("has more than one unspent credit")
	}
}

func newCoinBase(outputValues ...int64) *wire.MsgTx {
	tx := wire.MsgTx{
		TxIn: []*wire.TxIn{
			&wire.TxIn{
				PreviousOutPoint: wire.OutPoint{Index: ^uint32(0)},
			},
		},
	}
	for _, val := range outputValues {
		tx.TxOut = append(tx.TxOut, &wire.TxOut{Value: val})
	}
	return &tx
}

func spendOutput(txHash *chainhash.Hash, index uint32, outputValues ...int64) *wire.MsgTx {
	tx := wire.MsgTx{
		TxIn: []*wire.TxIn{
			&wire.TxIn{
				PreviousOutPoint: wire.OutPoint{Hash: *txHash, Index: index},
			},
		},
	}
	for _, val := range outputValues {
		tx.TxOut = append(tx.TxOut, &wire.TxOut{Value: val})
	}
	return &tx
}

func TestCoinbases(t *testing.T) {
	t.Parallel()

	s, teardown, err := testStore()
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	b100 := BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}

	cb := newCoinBase(20e8, 10e8, 30e8)
	cbRec, err := NewTxRecordFromMsgTx(cb, b100.Time)
	if err != nil {
		t.Fatal(err)
	}

	// Insert coinbase and mark outputs 0 and 2 as credits.
	err = s.InsertTx(cbRec, &b100)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(cbRec, &b100, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(cbRec, &b100, 2, false)
	if err != nil {
		t.Fatal(err)
	}

	// Balance should be 0 if the coinbase is immature, 50 DCR at and beyond
	// maturity.
	//
	// Outputs when depth is below maturity are never included, no matter
	// the required number of confirmations.  Matured outputs which have
	// greater depth than minConf are still excluded.
	type balTest struct {
		height  int32
		minConf int32
		bal     dcrutil.Amount
	}
	balTests := []balTest{
		// Next block it is still immature
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 2,
			minConf: 0,
			bal:     0,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 2,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity),
			bal:     0,
		},

		// Next block it matures
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: 0,
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity),
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 1,
			bal:     0,
		},

		// Matures at this block
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: 0,
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity),
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 2,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(tst.minConf, tst.height, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after inserting coinbase")
	}

	// Spend an output from the coinbase tx in an unmined transaction when
	// the next block will mature the coinbase.
	spenderATime := time.Now()
	spenderA := spendOutput(&cbRec.Hash, 0, 5e8, 15e8)
	spenderARec, err := NewTxRecordFromMsgTx(spenderA, spenderATime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(spenderARec, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(spenderARec, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	balTests = []balTest{
		// Next block it matures
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: 1,
			bal:     30e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity),
			bal:     30e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity) - 1,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 1,
			bal:     0,
		},

		// Matures at this block
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: 1,
			bal:     30e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity),
			bal:     30e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 1,
			bal:     30e8,
		},
		{
			height:  b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity),
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 2,
			bal:     0,
		},
	}
	balTestsBeforeMaturity := balTests
	for i, tst := range balTests {
		bal, err := s.Balance(tst.minConf, tst.height, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after spending coinbase with unmined transaction")
	}

	// Mine the spending transaction in the block the coinbase matures.
	bMaturity := BlockMeta{
		Block: Block{Height: b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity)},
		Time:  time.Now(),
	}
	err = s.InsertTx(spenderARec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}

	balTests = []balTest{
		// Maturity height
		{
			height:  bMaturity.Height,
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 1,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 2,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity),
			bal:     30e8,
		},
		{
			height:  bMaturity.Height,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 1,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 2,
			bal:     0,
		},

		// Next block after maturity height
		{
			height:  bMaturity.Height + 1,
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 2,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 3,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 2,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: int32(chaincfg.TestNetParams.CoinbaseMaturity) + 3,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(tst.minConf, tst.height, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks mining coinbase spending transaction")
	}

	// Create another spending transaction which spends the credit from the
	// first spender.  This will be used to test removing the entire
	// conflict chain when the coinbase is later reorged out.
	//
	// Use the same output amount as spender A and mark it as a credit.
	// This will mean the balance tests should report identical results.
	spenderBTime := time.Now()
	spenderB := spendOutput(&spenderARec.Hash, 0, 5e8)
	spenderBRec, err := NewTxRecordFromMsgTx(spenderB, spenderBTime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(spenderBRec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(spenderBRec, &bMaturity, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	for i, tst := range balTests {
		bal, err := s.Balance(tst.minConf, tst.height, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks mining second spending transaction")
	}

	// Reorg out the block that matured the coinbase and check balances
	// again.
	err = s.Rollback(bMaturity.Height)
	if err != nil {
		t.Fatal(err)
	}
	balTests = balTestsBeforeMaturity
	for i, tst := range balTests {
		bal, err := s.Balance(tst.minConf, tst.height, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after reorging maturity block")
	}

	// Reorg out the block which contained the coinbase.  There should be no
	// more transactions in the store (since the previous outputs referenced
	// by the spending tx no longer exist), and the balance will always be
	// zero.
	err = s.Rollback(b100.Height)
	if err != nil {
		t.Fatal(err)
	}
	balTests = []balTest{
		// Current height
		{
			height:  b100.Height - 1,
			minConf: 0,
			bal:     0,
		},
		{
			height:  b100.Height - 1,
			minConf: 1,
			bal:     0,
		},

		// Next height
		{
			height:  b100.Height,
			minConf: 0,
			bal:     0,
		},
		{
			height:  b100.Height,
			minConf: 1,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(tst.minConf, tst.height, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after reorging coinbase block")
	}
	unminedTxs, err := s.UnminedTxs()
	if err != nil {
		t.Fatal(err)
	}
	if len(unminedTxs) != 0 {
		t.Fatalf("Should have no unmined transactions after coinbase reorg, found %d", len(unminedTxs))
	}
}

// Test moving multiple transactions from unmined buckets to the same block.
func TestMoveMultipleToSameBlock(t *testing.T) {
	t.Parallel()

	s, teardown, err := testStore()
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	b100 := BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}

	cb := newCoinBase(20e8, 30e8)
	cbRec, err := NewTxRecordFromMsgTx(cb, b100.Time)
	if err != nil {
		t.Fatal(err)
	}

	// Insert coinbase and mark both outputs as credits.
	err = s.InsertTx(cbRec, &b100)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(cbRec, &b100, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(cbRec, &b100, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	// Create and insert two unmined transactions which spend both coinbase
	// outputs.
	spenderATime := time.Now()
	spenderA := spendOutput(&cbRec.Hash, 0, 1e8, 2e8, 18e8)
	spenderARec, err := NewTxRecordFromMsgTx(spenderA, spenderATime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(spenderARec, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(spenderARec, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(spenderARec, nil, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	spenderBTime := time.Now()
	spenderB := spendOutput(&cbRec.Hash, 1, 4e8, 8e8, 18e8)
	spenderBRec, err := NewTxRecordFromMsgTx(spenderB, spenderBTime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(spenderBRec, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(spenderBRec, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(spenderBRec, nil, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	// Mine both transactions in the block that matures the coinbase.
	bMaturity := BlockMeta{
		Block: Block{Height: b100.Height + int32(chaincfg.TestNetParams.CoinbaseMaturity)},
		Time:  time.Now(),
	}
	err = s.InsertTx(spenderARec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(spenderBRec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}

	// Check that both transactions can be queried at the maturity block.
	detailsA, err := s.UniqueTxDetails(&spenderARec.Hash, &bMaturity.Block)
	if err != nil {
		t.Fatal(err)
	}
	if detailsA == nil {
		t.Fatal("No details found for first spender")
	}
	detailsB, err := s.UniqueTxDetails(&spenderBRec.Hash, &bMaturity.Block)
	if err != nil {
		t.Fatal(err)
	}
	if detailsB == nil {
		t.Fatal("No details found for second spender")
	}

	// Verify that the balance was correctly updated on the block record
	// append and that no unmined transactions remain.
	balTests := []struct {
		height  int32
		minConf int32
		bal     dcrutil.Amount
	}{
		// Maturity height
		{
			height:  bMaturity.Height,
			minConf: 0,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 1,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 2,
			bal:     0,
		},

		// Next block after maturity height
		{
			height:  bMaturity.Height + 1,
			minConf: 0,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 2,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 3,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(tst.minConf, tst.height, wtxmgr.BFBalanceSpendable)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after moving both coinbase spenders")
	}
	unminedTxs, err := s.UnminedTxs()
	if err != nil {
		t.Fatal(err)
	}
	if len(unminedTxs) != 0 {
		t.Fatalf("Should have no unmined transactions mining both, found %d", len(unminedTxs))
	}
}

// Test the optional-ness of the serialized transaction in a TxRecord.
// NewTxRecord and NewTxRecordFromMsgTx both save the serialized transaction, so
// manually strip it out to test this code path.
func TestInsertUnserializedTx(t *testing.T) {
	t.Parallel()

	s, teardown, err := testStore()
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	tx := newCoinBase(50e8)
	rec, err := NewTxRecordFromMsgTx(tx, timeNow())
	if err != nil {
		t.Fatal(err)
	}
	b100 := makeBlockMeta(100)
	err = s.InsertTx(stripSerializedTx(rec), &b100)
	if err != nil {
		t.Fatalf("Insert for stripped TxRecord failed: %v", err)
	}

	// Ensure it can be retreived successfully.
	details, err := s.UniqueTxDetails(&rec.Hash, &b100.Block)
	if err != nil {
		t.Fatal(err)
	}
	rec2, err := NewTxRecordFromMsgTx(&details.MsgTx, rec.Received)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rec.SerializedTx, rec2.SerializedTx) {
		t.Fatal("Serialized txs for coinbase do not match")
	}

	// Now test that path with an unmined transaction.
	tx = spendOutput(&rec.Hash, 0, 50e8)
	rec, err = NewTxRecordFromMsgTx(tx, timeNow())
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	details, err = s.UniqueTxDetails(&rec.Hash, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec2, err = NewTxRecordFromMsgTx(&details.MsgTx, rec.Received)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rec.SerializedTx, rec2.SerializedTx) {
		t.Fatal("Serialized txs for coinbase spender do not match")
	}
}