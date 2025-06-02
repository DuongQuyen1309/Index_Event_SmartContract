// SPDX-License-Identifier: MIT
// An example of a consumer contract that also owns and manages the subscription
pragma solidity ^0.8.7;

import "LinkTokenInterface.sol";
import "VRFCoordinatorV2Interface.sol";
import "VRFConsumerBaseV2.sol";
import "Ownable.sol";
import "Pausable.sol";
import "Counters.sol";
import "IERC20.sol";
import "SafeERC20.sol";
import "ECDSA.sol";

// assume that Wheel contract has enough BirthdayTicket, GAFI tokens
contract WheelV2 is VRFConsumerBaseV2, Ownable, Pausable {
    using Counters for Counters.Counter;
    using SafeERC20 for IERC20;

    VRFCoordinatorV2Interface private COORDINATOR;
    LinkTokenInterface private LINK_TOKEN;
    IERC20 private GAFI_TOKEN;
    IERC20 private BIRTHDAY_TICKET_TOKEN;
    address public SIGNER;
    string private SIGNER_METHOD = "Wheel:buyTicketByFish";
    uint256 public LOCK_DURATION = 600; // 10 minutes
    uint256 public TIME_EPSILON = 120; // 2 minutes

    struct TicketTransaction {
        // transactionID
        uint256 timestamp;
        uint256 amount;
        address user;
        uint8 transactionType; // 0: GAFI, 1: FISH
    }

    struct UserRequest {
        uint256 amount;
        address user;
    }

    // The gas lane to use, which specifies the maximum gas price to bump to
    bytes32 private keyHash;

    // A reasonable default is 100000
    uint32 private callbackGasLimit = 400000;

    // The default is 3, but you can set this higher.
    uint16 private requestConfirmations = 3;

    Counters.Counter private _transactionIds;

    // subscription id
    uint64 public subscriptionId;
    uint256 public maxPerTurn;
    uint256 public ticketPriceByGAFI = 1 * 1e17;
    // 380, 250, 345, 5, 7, 3, 8, 2
    uint256[8] private chances = [380, 630, 975, 980, 987, 990, 998, 1000];
    uint256[8] private prizes = [0, 1e17, 1, 25e16, 2, 5e17, 15e16, 25e17];

    // requestID to user
    mapping (uint256 => address) public requestToUser;

    // request ID --> tickets
//    mapping (uint256 => Ticket) public tickets;
//    mapping (uint256 => uint256[]) public requestToTickets; //
//    mapping (address => uint256[]) private users; //
//    mapping (uint256 => uint256) private prizes;
    mapping (uint256 => TicketTransaction) public transactions;
    mapping (address => uint256[]) public transactionIds; //
    mapping (address => uint256) public lockingTicketByFish;

    event RequestCreated(address indexed user, uint256 indexed requestId, uint256 amount);
    event ResponseCreated(address indexed user, uint256 indexed requestId, uint256[] prizeIds);

    constructor(
        address _gafi,
        address _birthday_ticket,
        address _link,
        address _vrfCoordinator,
        bytes32 _keyHash
    ) VRFConsumerBaseV2(_vrfCoordinator) {
        GAFI_TOKEN = IERC20(_gafi);
        BIRTHDAY_TICKET_TOKEN = IERC20(_birthday_ticket);
        COORDINATOR = VRFCoordinatorV2Interface(_vrfCoordinator);
        LINK_TOKEN = LinkTokenInterface(_link);
        keyHash = _keyHash;

        //Create a new subscription when you deploy the contract.
        createNewSubscription();
        maxPerTurn = 10;
    }

    // Assumes the subscription is funded sufficiently.
    function spin(uint256 amount) external whenNotPaused {
        // don't allow contract to spin
        require(tx.origin == msg.sender, "Invalid user");
        require(amount <= maxPerTurn, "Invalid limit");

        // get fee - birthday token has decimal 0
        BIRTHDAY_TICKET_TOKEN.safeTransferFrom(msg.sender, address(this), amount);

        // Will revert if subscription is not set and funded.
        uint256 requestId = COORDINATOR.requestRandomWords(
            keyHash,
            subscriptionId,
            requestConfirmations,
            callbackGasLimit,
            uint32(amount)
        );

        require(requestToUser[requestId] == address(0), "Request ID is existed");
        requestToUser[requestId] = msg.sender;
        emit RequestCreated(msg.sender, requestId, amount);
    }

    function buyTicket(uint256 amount) external whenNotPaused {
        // don't allow contract to by ticket
        require(tx.origin == msg.sender, "Invalid user");
        require(amount > 0, "Invalid amount");

        GAFI_TOKEN.safeTransferFrom(msg.sender, address(this), amount * ticketPriceByGAFI);
        BIRTHDAY_TICKET_TOKEN.safeTransfer(msg.sender, amount);

        uint256 newTransactionId = _createGAFITransaction(msg.sender, amount);
        transactionIds[msg.sender].push(newTransactionId);
    }

    function buyTicketByFish(uint256 requestId, uint256 timestamp, uint256 amount, bytes memory signature) external whenNotPaused {
        // don't allow contract to by ticket
        require(tx.origin == msg.sender, "Invalid user");
        require(transactions[requestId].amount == 0 && requestId > 1e9, "Invalid requestId");
        require(amount > 0 && amount <= 100, "Invalid amount");
        require(block.timestamp >= timestamp - 5 && block.timestamp <= timestamp + TIME_EPSILON, "Invalid timestamp");
        require(verify(SIGNER, msg.sender, requestId, SIGNER_METHOD, timestamp, amount, signature), "Invalid signature");
        require(block.timestamp >= lockingTicketByFish[msg.sender], "Too many request");

        BIRTHDAY_TICKET_TOKEN.safeTransfer(msg.sender, amount);

        _createFishTransaction(requestId, msg.sender, timestamp, amount);
        transactionIds[msg.sender].push(requestId);
        lockingTicketByFish[msg.sender] = block.timestamp + LOCK_DURATION;
    }

    function verify(
        address _signer,
        address _user,
        uint256 _requestFishID,
        string memory _method,
        uint256 _timestamp,
        uint256 _amount,
        bytes memory signature
    ) public pure returns (bool) {
        bytes32 messageHash = keccak256(abi.encodePacked(_user, _requestFishID, _method, _timestamp, _amount));
        bytes32 ethSignedMessageHash = ECDSA.toEthSignedMessageHash(messageHash);

        return ECDSA.recover(ethSignedMessageHash, signature) == _signer;
    }

    function fulfillRandomWords(
        uint256 requestId,
        uint256[] memory randomWords
    ) internal override {
        uint256 length = randomWords.length;
        uint256 totalGAFI = 0;
        uint256 totalTickets = 0;
        address user = requestToUser[requestId];
        require(user != address(0), "Invalid requestId");
        uint256[] memory prizeIds = new uint256[](length);
        uint256[8] memory _chances = chances;
        uint256[8] memory _prize = prizes;

        for (uint256 index = 0; index < length; index = unsafe_inc(index)) {
            uint256 prizeId = _random(randomWords[index], _chances);
            // no reward
            if (prizeId == 0) {
                continue;
            }
            prizeIds[index] = prizeId;

            // tickets
            if (prizeId == 2 || prizeId == 4) {
                totalTickets = totalTickets + _prize[prizeId];
                continue;
            }

            // default reward is GAFI
            totalGAFI = totalGAFI + _prize[prizeId];
        }

        if (totalTickets > 0) {
            BIRTHDAY_TICKET_TOKEN.safeTransfer(user, totalTickets);
        }

        if (totalGAFI > 0) {
            GAFI_TOKEN.safeTransfer(user, totalGAFI);
        }

        emit ResponseCreated(user, requestId, prizeIds);
    }

    function _random(uint256 seed, uint256[8] memory _chances) private pure returns (uint256) {
        uint256 value = seed % 1000;
        if (value < _chances[0]) {
            return 0;
        }

        if (value < _chances[1]) {
            return 1;
        }

        if (value < _chances[2]) {
            return 2;
        }

        if (value < _chances[3]) {
            return 3;
        }

        if (value < _chances[4]) {
            return 4;
        }

        if (value < _chances[5]) {
            return 5;
        }

        if (value < _chances[6]) {
            return 6;
        }

        if (value < _chances[7]) {
            return 7;
        }

        return 0;
    }

    function _createGAFITransaction(address user, uint256 numberOfTickets) private returns (uint256){
        _transactionIds.increment();
        uint256 newTransactionId = _transactionIds.current();
        require(transactions[newTransactionId].amount == 0, "buy ticket error");

        // timestamp, amount, user, type
        transactions[newTransactionId] = TicketTransaction(block.timestamp, numberOfTickets, user, 0);
        return newTransactionId;
    }

    function _createFishTransaction(uint256 requestId, address user, uint256 timestamp, uint256 numberOfTickets) private {
        // timestamp, amount, user, type
        transactions[requestId] = TicketTransaction(timestamp, numberOfTickets, user, 1);
    }

    // Create a new subscription when the contract is initially deployed.
    function createNewSubscription() private onlyOwner {
        subscriptionId = COORDINATOR.createSubscription();
        // Add this contract as a consumer of its own subscription.
        COORDINATOR.addConsumer(subscriptionId, address(this));
    }

    // Assumes this contract owns link.
    function topUpSubscription() external onlyOwner {
        LINK_TOKEN.transferAndCall(address(COORDINATOR), LINK_TOKEN.balanceOf(address(this)), abi.encode(subscriptionId));
    }

    function addConsumer(address consumerAddress) external onlyOwner {
        // Add a consumer contract to the subscription.
        COORDINATOR.addConsumer(subscriptionId, consumerAddress);
    }

    function removeConsumer(address consumerAddress) external onlyOwner {
        // Remove a consumer contract from the subscription.
        COORDINATOR.removeConsumer(subscriptionId, consumerAddress);
    }

    function cancelSubscription(address receivingWallet) external onlyOwner {
        // Cancel the subscription and send the remaining LINK to a wallet address.
        COORDINATOR.cancelSubscription(subscriptionId, receivingWallet);
        subscriptionId = 0;
    }

    // Transfer this contract's funds to an address.
    function withdrawLink(address to, uint256 amount) external onlyOwner {
        LINK_TOKEN.transfer(to, amount);
    }

    // Emergency methods
    function emergencyWithdraw(address token, uint256 amount) external onlyOwner {
        IERC20(token).safeTransfer(msg.sender, amount);
    }

    function emergencyWithdrawAll() external onlyOwner {
        GAFI_TOKEN.safeTransfer(msg.sender, GAFI_TOKEN.balanceOf(address(this)));
        BIRTHDAY_TICKET_TOKEN.safeTransfer(msg.sender, BIRTHDAY_TICKET_TOKEN.balanceOf(address(this)));
    }

    function changeKeyHash(bytes32 _keyHash) external onlyOwner {
        keyHash = _keyHash;
    }

    function changeGas(uint32 _gas) external onlyOwner {
        callbackGasLimit = _gas;
    }

    function changeTicketPriceByGAFI(uint256 price) external onlyOwner {
        ticketPriceByGAFI = price;
    }

    function changePauseState(bool state) external onlyOwner {
        if (state) {
            _pause();
            return;
        }

        _unpause();
    }

    function changeTimeEpsilon(uint256 epsilon) external onlyOwner {
        TIME_EPSILON = epsilon;
    }

    function changeSigner(address signer) external onlyOwner {
        SIGNER = signer;
    }

    function changeLockDuration(uint256 lockDuration) external onlyOwner {
        LOCK_DURATION = lockDuration;
    }

    function fulfillRandomWordsIfChainlinkFail(
        uint256 requestId,
        uint256[] memory randomWords
    ) external onlyOwner {
        fulfillRandomWords(requestId, randomWords);
    }

    function unsafe_inc(uint256 x) private pure returns (uint256) {
        unchecked { return x + 1; }
    }
}