package myevm

func TEvm() {
	// Create a new state database for the EVM.
	// stateDB, _ := state.New(common.Hash{}, state.NewDatabase(nil), &snapshot.Tree{})

	// // Create a new EVM instance.
	// evm := vm.NewEVM(context.Background(), nil, stateDB, nil, nil)

	// // Deploy a simple smart contract.
	// contractBytecode := common.FromHex("608060405234801561001057600080fd5b50610151806100206000396000f3fe")
	// sender := crypto.PubkeyToAddress(ecdsa.PublicKey{})
	// value := big.NewInt(0)
	// gasLimit := uint64(300000)
	// contractAddr, _, contract, err := evm.Create(sender, contractBytecode, value, gasLimit)
	// if err != nil {
	// 	log.Fatalf("Failed to create contract: %v", err)
	// }
	// fmt.Printf("Contract deployed at address: %s\n", contractAddr.Hex())

	// // Interact with the deployed contract.
	// inputData := common.Hex2Bytes("60fe47b1000000000000000000000000000000000000000000000000000000000000000a")
	// result, leftOverGas, err := evm.Call(context.Background(), types.CallMsg{
	// 	From:     sender,
	// 	To:       &contractAddr,
	// 	Data:     inputData,
	// 	Gas:      gasLimit,
	// 	GasPrice: big.NewInt(1),
	// 	Value:    big.NewInt(0),
	// })
	// if err != nil {
	// 	log.Fatalf("Failed to call contract: %v", err)
	// }
	// fmt.Printf("Contract call result: %s\n", string(result))
	// fmt.Printf("Contract call leftOverGas: %s\n", string(leftOverGas))
}
