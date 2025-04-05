module my_address::btc_tokenv3 {
    use std::error;
    use std::signer;
    use std::string::{Self, String};
    use aptos_framework::coin::{Self, BurnCapability, FreezeCapability, MintCapability};
    use aptos_framework::event;
    use aptos_framework::account;
    use std::option;

    /// BTC token on Aptos representing cross-chain Bitcoin
    struct BTC {}
    
    /// Storing the mint, burn, freeze capabilities for the BTC token
    struct BTCCapabilities has key {
        mint_cap: MintCapability<BTC>,
        burn_cap: BurnCapability<BTC>,
        freeze_cap: FreezeCapability<BTC>,
    }

    /// Events for tracking bridge operations
    struct BridgeEvents has key {
        mint_events: event::EventHandle<MintEvent>,
        burn_events: event::EventHandle<BurnEvent>,
    }

    /// Event emitted when BTC is minted on Aptos
    struct MintEvent has drop, store {
        amount: u64,
        recipient: address,
        btc_txid: String,  // Bitcoin transaction ID
    }

    /// Event emitted when BTC is burned on Aptos for withdrawal to Bitcoin
    struct BurnEvent has drop, store {
        amount: u64,
        burner: address,
        btc_address: String,  // Bitcoin address for receiving funds
    }

    /// Error codes
    const E_NOT_AUTHORIZED: u64 = 1;
    const E_NOT_FOUND: u64 = 2;
    const E_ALREADY_INITIALIZED: u64 = 3;
    const E_INSUFFICIENT_BALANCE: u64 = 4;
    const E_NOT_IMPLEMENTED: u64 = 5; // Used for functions that aren't fully implemented yet
    const E_AMOUNT_SMALLER_THAN_FEE: u64 = 6;
    const E_MAX_SUPPLY_EXCEEDED: u64 = 7;
    const E_INVALID_RECIPIENT: u64 = 8;

    /// Constants
    const MAX_BTC_SUPPLY: u128 = 21000000 * 100000000; // 21 million BTC with 8 decimal places (u128 to handle large values)

    /// Get the admin address that controls the token
    fun get_admin_address(): address {
        @my_address
    }
    
    /// Get the bridge fee amount
    fun get_fee(): u64 {
        // Fixed fee of 0.0001 BTC (10000 satoshis)
        10000
    }
    
    /// Initialize the module with token capabilities
    public entry fun initialize_module(account: &signer) {
        let account_addr = signer::address_of(account);
        // Only admin can initialize
        assert!(account_addr == get_admin_address(), error::permission_denied(E_NOT_AUTHORIZED));
        
        // Initialize only once
        assert!(!exists<BTCCapabilities>(account_addr), error::already_exists(E_ALREADY_INITIALIZED));
        
        let (mint_cap, burn_cap, freeze_cap) = initialize(account);
        store_capabilities(account, mint_cap, burn_cap, freeze_cap);
        // Register the admin account to receive BTC
        // registerv2(account,account);
        register(account);
    }

    /// Initialize the BTC token and return capabilities
    public fun initialize(account: &signer): (MintCapability<BTC>, BurnCapability<BTC>, FreezeCapability<BTC>) {
        let account_addr = signer::address_of(account);
        
        // Create the BTC token
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<BTC>(
            account,
            string::utf8(b"Bitcoin"),
            string::utf8(b"BTC"),
            8, // BTC has 8 decimal places
            true, // Monitor supply
        );

        // Register the events
        if (!exists<BridgeEvents>(account_addr)) {
            move_to(account, BridgeEvents {
                mint_events: account::new_event_handle<MintEvent>(account),
                burn_events: account::new_event_handle<BurnEvent>(account),
            });
        };
        
        // Return capabilities in the correct order expected by the storage function
        (mint_cap, burn_cap, freeze_cap)
    }

    /// Save the capabilities in the module creator's account
    public fun store_capabilities(
        account: &signer,
        mint_cap: MintCapability<BTC>,
        burn_cap: BurnCapability<BTC>,
        freeze_cap: FreezeCapability<BTC>
    ) {
        let account_addr = signer::address_of(account);
        
        // Ensure capabilities haven't been stored already
        assert!(!exists<BTCCapabilities>(account_addr), error::already_exists(E_ALREADY_INITIALIZED));
        
        // Store the capabilities
        move_to(account, BTCCapabilities {
            mint_cap,
            burn_cap,
            freeze_cap,
        });
    }

    /// Register an account to receive BTC
    public entry fun register(account: &signer) {
        coin::register<BTC>(account);
    }

    // public entry fun registerv2(admin: &signer, account: &signer) {
    //     let admin_addr = signer::address_of(admin);
    //     let account_addr = signer::address_of(account);
    //     assert!(admin_addr == get_admin_address(), error::permission_denied(E_NOT_AUTHORIZED));
    //     assert!(account_addr != @0x0, error::invalid_argument(E_INVALID_RECIPIENT));
    //     assert!(coin::is_account_registered<BTC>(account_addr), error::invalid_argument(E_ALREADY_INITIALIZED));
        
    //     coin::register<BTC>(account);
    // }


    /// Mint BTC tokens and transfer to the receiver
    public entry fun mint_tokens(
        admin: &signer,
        recipient: address,
        amount: u64,
        btc_txid: String,
    ) acquires BTCCapabilities, BridgeEvents {
        let admin_addr = signer::address_of(admin);
        
        // Verify admin has minting capability
        assert!(admin_addr == get_admin_address(), error::permission_denied(E_NOT_AUTHORIZED));
        assert!(exists<BTCCapabilities>(admin_addr), error::not_found(E_NOT_FOUND));
        
        // Get minting capability
        let capabilities = borrow_global<BTCCapabilities>(admin_addr);
        
        // coin::register<BTC>(recipient);

        // Check if recipient is valid
        // assert!(recipient != @0x0, error::invalid_argument(E_INVALID_RECIPIENT));

        // Get the bridge fee
        let fee = 0;
        assert!(amount > fee, error::invalid_argument(E_AMOUNT_SMALLER_THAN_FEE));

        // Check if total supply would exceed max supply
        let current_supply = coin::supply<BTC>();
        if (option::is_some(&current_supply)) {
            let supply = *option::borrow(&current_supply);
            assert!((supply as u128) + (amount as u128) <= MAX_BTC_SUPPLY, error::invalid_argument(E_MAX_SUPPLY_EXCEEDED));
        };

        // Make sure recipient is registered
        if (!coin::is_account_registered<BTC>(recipient)) {
            // If using a real contract, we'd need proper registration
            // For now, just abort with proper error
            abort error::invalid_argument(E_ALREADY_INITIALIZED)
        };
        
        // Mint tokens - recipient gets amount minus fee
        let recipient_amount = amount - fee;
        let recipient_coins = coin::mint<BTC>(recipient_amount, &capabilities.mint_cap);
        coin::deposit<BTC>(recipient, recipient_coins);
        
        // Mint fee tokens to admin
        let fee_coins = coin::mint<BTC>(fee, &capabilities.mint_cap);
        coin::deposit<BTC>(admin_addr, fee_coins);
        
        // Emit mint event
        let events = borrow_global_mut<BridgeEvents>(admin_addr);
        event::emit_event(&mut events.mint_events, MintEvent {
            amount: recipient_amount,
            recipient,
            btc_txid,
        });
    }

    /// Burn BTC tokens and emit appropriate event, simulating withdrawal to BTC address
    public fun burn_from(
        account_addr: address, 
        amount: u64, 
        btc_receiver_address: String
    ) acquires BTCCapabilities, BridgeEvents {
        let admin_addr = get_admin_address();
        let capabilities = borrow_global<BTCCapabilities>(admin_addr);
        
        // Check if user has enough balance
        let balance = coin::balance<BTC>(account_addr);
        assert!(balance >= amount, error::invalid_argument(E_INSUFFICIENT_BALANCE));
        // Burn the remaining coins
        let coins_to_burn = coin::mint<BTC>(amount, &capabilities.mint_cap);
        coin::burn(coins_to_burn, &capabilities.burn_cap);
        
        // Emit burn event
        let events = borrow_global_mut<BridgeEvents>(admin_addr);
        event::emit_event(&mut events.burn_events, BurnEvent {
            amount,
            burner: account_addr,
            btc_address: btc_receiver_address,
        });
    }



    public entry fun transfer(from: &signer, to: address, amount: u64) {
        coin::transfer<BTC>(from, to, amount);
    }
    
    /// Get the balance of BTC for an account
    public fun balance(addr: address): u64 {
        coin::balance<BTC>(addr)
    }

    /// Get the total supply of BTC
    public fun total_supply(): u64 {
        let supply_opt = coin::supply<BTC>();
        if (option::is_some(&supply_opt)) {
            let u128_supply = *option::borrow(&supply_opt);
            (u128_supply as u64)
        } else {
            0
        }
    }
}
