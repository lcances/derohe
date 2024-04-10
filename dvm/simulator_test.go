// Copyright 2017-2018 DERO Project. All rights reserved.
// Use of this source code in any form is governed by RESEARCH license.
// license can be found in the LICENSE file.
// GPG: 0F39 E425 8C65 3947 702A  8234 08B2 0360 A03A 9DE8
//
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL
// THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF
// THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package dvm

//import "fmt"
//import "reflect"
import "strings"
import "testing"

import "github.com/deroproject/derohe/rpc"
import "github.com/deroproject/derohe/cryptography/crypto"

var sc = `/* Lottery Smart Contract Example in DVM-BASIC.  
This lottery smart contract will give lottery wins on every second try in following default contract.
	Make depost transaction to this SCID to play lottery. 
	Check https://github.com/deroproject/derohe/blob/main/guide/examples/lottery_sc_guide.md
*/



        Function Lottery() Uint64
	10  dim deposit_count,winner as Uint64
	20  LET deposit_count =  LOAD("deposit_count")+1
	25  IF DEROVALUE() == 0 THEN GOTO 110  // if deposit amount is 0, simply return
	30  STORE("depositor_address" + (deposit_count-1), SIGNER()) // store address for later on payment
	40  STORE("deposit_total", LOAD("deposit_total") + DEROVALUE() )
	50  STORE("deposit_count",deposit_count)
	60  IF LOAD("lotteryeveryXdeposit") > deposit_count THEN GOTO 110 // we will wait till X players join in
        // we are here means all players have joined in, roll the DICE, 
	70  LET winner  = RANDOM() % deposit_count // we have a winner
	80  SEND_DERO_TO_ADDRESS(LOAD("depositor_address" + winner) , LOAD("lotterygiveback")*LOAD("deposit_total")/10000)
        // Re-Initialize for another round
        90  STORE("deposit_count", 0)   //  initial players
	100 STORE("deposit_total", 0)   //  total deposit of all players
	110  RETURN 0
	End Function

	
	// This function is used to initialize parameters during install time
	Function Initialize() Uint64
	5   version("1.2.3")
	10  STORE("owner", SIGNER())   // store in DB  ["owner"] = address
	20  STORE("lotteryeveryXdeposit", 2)   // lottery will reward every X deposits
        // How much will lottery giveback in 1/10000 parts, granularity .01 %
	30  STORE("lotterygiveback", 9900)   // lottery will give reward 99% of deposits, 1 % is accumulated for owner to withdraw
	33  STORE("deposit_count", 0)   //  initial players
	34  STORE("deposit_total", 0)   //  total deposit of all players
	// 35 printf "Initialize executed"
	40 RETURN 0 
	End Function 
	
	
	
        // Function to tune lottery parameters
	Function TuneLotteryParameters(input Uint64, lotteryeveryXdeposit Uint64, lotterygiveback Uint64) Uint64
	10  dim key,stored_owner as String
	20  dim value_uint64 as Uint64
	30  IF LOAD("owner") == SIGNER() THEN GOTO 100  // check whether owner is real owner
	40  RETURN 1
	
	100  STORE("lotteryeveryXdeposit", lotteryeveryXdeposit)   // lottery will reward every X deposits
	130  STORE("lotterygiveback", value_uint64)   // how much will lottery giveback in 1/10000 parts, granularity .01 %
	140  RETURN 0 // return success
	End Function
	

	
	// This function is used to change owner 
	// owner is an string form of address 
	Function TransferOwnership(newowner String) Uint64 
	10  IF LOAD("owner") == SIGNER() THEN GOTO 30 
	20  RETURN 1
	30  STORE("tmpowner",ADDRESS_RAW(newowner))
	40  RETURN 0
	End Function
	
	// Until the new owner claims ownership, existing owner remains owner
        Function ClaimOwnership() Uint64 
	10  IF LOAD("tmpowner") == SIGNER() THEN GOTO 30 
	20  RETURN 1
	30  STORE("owner",SIGNER()) // ownership claim successful
	40  RETURN 0
	End Function
	
	// If signer is owner, withdraw any requested funds
	// If everthing is okay, they will be showing in signers wallet
        Function Withdraw( amount Uint64) Uint64 
	10  IF LOAD("owner") == SIGNER() THEN GOTO 30 
	20  RETURN 1
	30  SEND_DERO_TO_ADDRESS(SIGNER(),amount)
	40  RETURN 0
	End Function
	
	// If signer is owner, provide him rights to update code anytime
        // make sure update is always available to SC
        Function UpdateCode( code String) Uint64 
	10  IF LOAD("owner") == SIGNER() THEN GOTO 30 
	20  RETURN 1
	30  UPDATE_SC_CODE(code)
	40  RETURN 0
	End Function
	`

// run the test
func initializeTest(code string) (*Simulator, *rpc.Address, crypto.Hash, uint64, uint64, error) {
	s := SimulatorInitialize(nil)
	var addr *rpc.Address
	var err error
	var zerohash crypto.Hash

	if addr, err = rpc.NewAddress(strings.TrimSpace("deto1qy0ehnqjpr0wxqnknyc66du2fsxyktppkr8m8e6jvplp954klfjz2qqdzcd8p")); err != nil {
		return nil, nil, crypto.Hash{}, 0, 0, err
	}

	s.AccountAddBalance(*addr, zerohash, 500)
	scid, gascompute, gasstorage, err := s.SCInstall(code, map[crypto.Hash]uint64{}, rpc.Arguments{}, addr, 0)

	if err != nil {
		return nil, nil, crypto.Hash{}, 0, 0, err
	}

	return s, addr, scid, gascompute, gasstorage, nil
}

func Test_Simulator_execution(t *testing.T) {
	s, addr, scid, gascompute, gasstorage, err := initializeTest(sc)
	var zerohash crypto.Hash

	if err != nil {
		t.Fatalf("cannot initialize test %s\n", err)
	}

	// trigger first time lottery play
	gascompute, gasstorage, err = s.RunSC(map[crypto.Hash]uint64{zerohash: 45}, rpc.Arguments{{rpc.SCACTION, rpc.DataUint64, uint64(rpc.SC_CALL)}, {rpc.SCID, rpc.DataHash, scid}, rpc.Argument{"entrypoint", rpc.DataString, "Lottery"}}, addr, 0)
	if err != nil {
		t.Fatalf("cannot run contract %s\n", err)
	}

	w_sc_data_tree := Wrapped_tree(s.cache, s.ss, scid)
	stored_value, _ := LoadSCAssetValue(w_sc_data_tree, scid, zerohash)

	if 45 != stored_value {
		t.Fatalf("storage corruption dero value")
	}

	if uint64(45) != ReadSCValue(w_sc_data_tree, scid, "deposit_total") {
		t.Fatalf("storage corruption")
	}

	// trigger second time lottery play
	gascompute, gasstorage, err = s.RunSC(map[crypto.Hash]uint64{zerohash: 55}, rpc.Arguments{{rpc.SCACTION, rpc.DataUint64, uint64(rpc.SC_CALL)}, {rpc.SCID, rpc.DataHash, scid}, rpc.Argument{"entrypoint", rpc.DataString, "Lottery"}}, addr, 0)
	if err != nil {
		t.Fatalf("cannot run contract %s\n", err)
	}

	w_sc_data_tree = Wrapped_tree(s.cache, s.ss, scid)

	// only 1 is left ( which is profit)
	if stored_value, _ = LoadSCAssetValue(w_sc_data_tree, scid, zerohash); 1 != stored_value {
		t.Fatalf("storage corruption dero value")
	}

	// total deposit must be zero
	if uint64(0) != ReadSCValue(w_sc_data_tree, scid, "deposit_total") {
		t.Fatalf("storage corruption")
	}

	_ = gascompute
	_ = gasstorage
}

var sc2 = `/* Minimal smart contract template in DVM-BASIC */
	Function Initialize() Uint64
	1 IF EXISTS("owner") THEN GOTO 10
	2 STORE("owner", SIGNER())
	3 STORE("original_owner", SIGNER())
	10 RETURN 0
	End Function

	Function UpdateCode(code String) Uint64
	1  IF LOAD("owner") == SIGNER() THEN GOTO 3
	2  RETURN 1
	3  UPDATE_SC_CODE(code)
	4  RETURN 0
	End Function

	Function AppendCode(code String) Uint64
	1  IF LOAD("owner") == SIGNER() THEN GOTO 3
	2  RETURN 1
	3  APPEND_SC_CODE(code)
	4  RETURN 0
	End Function
	`

var codeToAppend1 = `// This is the code to append
	Function CallRandom() Uint64
	1  RANDOM()
	2  RETURN 0
	End Function
	`

var codeToAppend2 = `// This is the code to append
	Function CallRandom2() Uint64
	1  RANDOM()
	2  RETURN 0
	End Function
	`

var makeImmutable = `// This is the code to append
	Function UpdateCode(code String) Uint64
	1  RETURN 0
	End Function

	Function AppendCode(code String) Uint64
	1 RETURN 0
	End Function
	`

func Test_SC_Changes(t *testing.T) {
	s, addr, scid, gascompute, gasstorage, err := initializeTest(sc2)
	// var zerohash crypto.Hash

	if err != nil {
		t.Fatalf("cannot initialize test %s\n", err)
	}

	// Call the AppendCode function
	gascompute, gasstorage, err = s.RunSC(map[crypto.Hash]uint64{}, rpc.Arguments{{rpc.SCACTION, rpc.DataUint64, uint64(rpc.SC_CALL)}, {rpc.SCID, rpc.DataHash, scid}, rpc.Argument{"entrypoint", rpc.DataString, "AppendCode"}, rpc.Argument{"code", rpc.DataString, codeToAppend1}}, addr, 0)
	if err != nil {
		t.Fatalf("cannot run contract %s\n", err)
	}

	// Check the new code to see if the function CallRandom exists
	w_sc_data_tree := Wrapped_tree(s.cache, s.ss, scid)
	sc_bytes, err := w_sc_data_tree.Get(SC_Code_Key(scid))
	if err != nil {
		t.Fatalf("cannot read code from SC %s\n", err)
	}
	// compare the code
	if !strings.Contains(string(sc_bytes), "Function CallRandom() Uint64") {
		t.Fatalf("AppendCode did not append the code correctly\n")
	}

	// Call the newly added function CallRandom
	gascompute, gasstorage, err = s.RunSC(map[crypto.Hash]uint64{}, rpc.Arguments{{rpc.SCACTION, rpc.DataUint64, uint64(rpc.SC_CALL)}, {rpc.SCID, rpc.DataHash, scid}, rpc.Argument{"entrypoint", rpc.DataString, "CallRandom"}}, addr, 0)
	if err != nil {
		t.Fatalf("cannot run contract %s\n", err)
	}

	// Make the SC immutable
	gascompute, gasstorage, err = s.RunSC(map[crypto.Hash]uint64{}, rpc.Arguments{{rpc.SCACTION, rpc.DataUint64, uint64(rpc.SC_CALL)}, {rpc.SCID, rpc.DataHash, scid}, rpc.Argument{"entrypoint", rpc.DataString, "UpdateCode"}, rpc.Argument{"code", rpc.DataString, makeImmutable}}, addr, 0)
	if err != nil {
		t.Fatalf("cannot run contract %s\n", err)
	}

	// Running AppendCode should not change the SC code
	gascompute, gasstorage, err = s.RunSC(map[crypto.Hash]uint64{}, rpc.Arguments{{rpc.SCACTION, rpc.DataUint64, uint64(rpc.SC_CALL)}, {rpc.SCID, rpc.DataHash, scid}, rpc.Argument{"entrypoint", rpc.DataString, "AppendCode"}, rpc.Argument{"code", rpc.DataString, codeToAppend2}}, addr, 0)
	if err != nil {
		t.Fatalf("cannot run contract %s\n", err)
	}

	// Check the new code to see if the function CallRandom2 exists (should fail)
	w_sc_data_tree = Wrapped_tree(s.cache, s.ss, scid)
	sc_bytes, err = w_sc_data_tree.Get(SC_Code_Key(scid))
	if err != nil {
		t.Fatalf("cannot read code from SC %s\n", err)
	}
	// compare the code
	if strings.Contains(string(sc_bytes), "Function CallRandom2() Uint64") {
		t.Fatalf("AppendCode should not append the code after the SC is made immutable\n")
	}

	_ = gascompute
	_ = gasstorage
}
