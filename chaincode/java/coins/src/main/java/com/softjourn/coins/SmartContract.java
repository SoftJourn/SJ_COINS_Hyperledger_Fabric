package com.softjourn.coins;

import com.softjourn.coins.dto.ExpirableTransaction;
import com.softjourn.coins.dto.TransferRequest;
import com.softjourn.coins.dto.UserBalance;
import com.softjourn.common.helper.ContextHelper;
import com.softjourn.common.helper.IdentityHelper;
import com.softjourn.common.helper.ObjectConverter;
import java.util.HashMap;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.function.Predicate;
import java.util.stream.Stream;
import org.hyperledger.fabric.Logger;
import org.hyperledger.fabric.contract.Context;
import org.hyperledger.fabric.contract.ContractInterface;
import org.hyperledger.fabric.contract.annotation.Contact;
import org.hyperledger.fabric.contract.annotation.Contract;
import org.hyperledger.fabric.contract.annotation.Default;
import org.hyperledger.fabric.contract.annotation.Info;
import org.hyperledger.fabric.contract.annotation.License;
import org.hyperledger.fabric.contract.annotation.Transaction;
import org.hyperledger.fabric.contract.annotation.Transaction.TYPE;
import org.hyperledger.fabric.shim.ChaincodeException;
import org.hyperledger.fabric.shim.ChaincodeStub;

@Default
@Contract(
    name = "coins",
    info = @Info(
        title = "SJCoins",
        description = "Softjourn coins transfer",
        version = "1.0",
        license = @License(
            name = "Apache 2.0 License",
            url = "http://www.apache.org/licenses/LICENSE-2.0.html"),
        contact = @Contact(email = "vzaichuk@softjourn.com", name = "Vladyslav Zaichuk")))
public class SmartContract implements ContractInterface {

  private static final String CURRENCY_KEY = "currency";
  private static final String MINTER_KEY = "minter";
  private static final String USER_ACCOUNT_TYPE = "user_";
  private static final String PROJECT_ACCOUNT_TYPE = "project_";
  private static final String BALANCE_MAP_KEY = "balances";
  private static final String EXPIRABLE_TRANSACTION_MAP_KEY = "expirable_transactions";
  private static final int TX_EXPIRATION_PERIOD = 3600;

  private final Logger logger = Logger.getLogger(getClass());

  private IdentityHelper identityHelper = null;
  private ContextHelper contextHelper = null;
  private ObjectConverter objectConverter = null;

  public SmartContract() {
  }

  public SmartContract(ContextHelper contextHelper, IdentityHelper identityHelper) {
    this.contextHelper = contextHelper;
    this.identityHelper = identityHelper;
  }

  /**
   * Init ledger with some start values.
   *
   * @param ctx The transaction context.
   */
  @Transaction
  public String initLedger(final Context ctx) {
    ChaincodeStub stub = ctx.getStub();

    List<String> parameters = stub.getParameters();
    if (parameters.size() != 2) {
      throw new ChaincodeException(
          String.format("Incorrect number of arguments. Expected 2, was %d", parameters.size()));
    }

    String currencyName = parameters.get(1); // TODO: Add validation of value.
    logger.info("_____ Init " + currencyName + "_____");
    stub.putState(CURRENCY_KEY, currencyName.getBytes());

    logger.info("Minter ID: " + parameters.get(0));
    byte[] minterBytes = parameters.get(0).getBytes();
    stub.putState(MINTER_KEY, minterBytes);

    IdentityHelper identityHelper = getIdentityHelper();
    String currentUserId = identityHelper.getCurrentUserId(ctx);
    String currentUserAccount = identityHelper
        .getUserAccount(ctx, USER_ACCOUNT_TYPE, currentUserId);
    logger.info("CurrentUserAccount: " + currentUserAccount);

    ContextHelper contextHelper = getContextHelper();
    // Init account balances map.
    Map<String, Long> balanceMap =
        (Map<String, Long>) contextHelper.getMap(ctx, BALANCE_MAP_KEY);
    if (balanceMap == null) {
      balanceMap = new HashMap<>();
      contextHelper.writeMap(ctx, BALANCE_MAP_KEY, balanceMap);
    }

    // Init expirable transaction map.
    Map<String, List<ExpirableTransaction>> expirableTransactionsMap =
        (Map<String, List<ExpirableTransaction>>)
            contextHelper.getMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY);
    if (expirableTransactionsMap == null) {
      expirableTransactionsMap = new HashMap<>();
      contextHelper.writeMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY, expirableTransactionsMap);
    }

    return currencyName;
  }

  /**
   * Mint coins to treasury account.
   *
   * @param ctx CC context.
   * @param amount Amount to mint.
   * @return Treasury account balance.
   */
  @Transaction
  public UserBalance mint(final Context ctx, final long amount) {
    logger.info("Mint amount: " + amount);
    if (amount < 1) {
      throw new ChaincodeException("Mint amount can't be negative or zero");
    }

    ContextHelper contextHelper = getContextHelper();
    String minterId = contextHelper.getState(ctx, MINTER_KEY, String.class);
    logger.info("Minter id: " + minterId);

    IdentityHelper identityHelper = getIdentityHelper();
    String currentUserId = identityHelper.getCurrentUserId(ctx);
    if (!currentUserId.equals(minterId)) {
      throw new ChaincodeException("Permission is denied for this operation");
    }

    String currentUserAccount =
        identityHelper.getUserAccount(ctx, USER_ACCOUNT_TYPE, currentUserId);
    logger.info("Current user account: " + currentUserAccount);

    Map<String, Long> balanceMap =
        (Map<String, Long>) contextHelper.getMap(ctx, BALANCE_MAP_KEY);
    balanceMap.putIfAbsent(currentUserAccount, 0L);
    balanceMap.put(currentUserAccount, balanceMap.get(currentUserAccount) + amount);

    contextHelper.writeMap(ctx, BALANCE_MAP_KEY, balanceMap);

    return new UserBalance(currentUserId, balanceMap.get(currentUserAccount));
  }

  /**
   * Transfer coins between accounts.
   *
   * @param ctx CC context.
   * @param receiverAccountType Receiver account type.
   * @param receiver Receiver account.
   * @param amount Amount to transfer.
   * @param expirable Expirable transfer flag.
   * @return Balance of donor account.
   */
  @Transaction
  public UserBalance transfer(
      final Context ctx, final String receiverAccountType, final String receiver,
      final long amount, final boolean expirable
  ) {
    logger.info("AccountType: " + receiverAccountType);
    logger.info("Receiver: " + receiver);
    logger.info("Amount: " + amount);

    if (amount < 1) {
      throw new ChaincodeException("Incorrect amount. Amount should be positive.");
    }

    IdentityHelper identityHelper = getIdentityHelper();
    String currentUserId = identityHelper.getCurrentUserId(ctx);

    String currentUserAccount = identityHelper
        .getUserAccount(ctx, USER_ACCOUNT_TYPE, currentUserId);
    logger.info("CurrentUserAccount: " + currentUserAccount);

    String receiverAccount = identityHelper.getUserAccount(ctx, receiverAccountType, receiver);
    logger.info("ReceiverAccount: " + receiverAccount);

    flushExpiredTransactions(ctx, currentUserAccount);
    flushExpiredTransactions(ctx, receiverAccount);
    UserBalance userBalance = getBalance(ctx, currentUserAccount, currentUserId);
    if (userBalance.getBalance() < amount) {
      throw new ChaincodeException("Not enough coins");
    }

    decreaseAccountBalance(ctx, currentUserAccount, amount);
    if (expirable) {
      addExpirableTransaction(
          ctx, receiverAccount, contextHelper.getNextId(),
          amount, contextHelper.getCurrentTimestamp());
    } else {
      changeBalance(ctx, receiverAccount, amount);
    }

    // Ledger state is cached via context helper, so reading balance is completely appropriate.
    return getBalance(ctx, currentUserAccount, currentUserId);
  }

  // TODO: This method doesn't contain access rule check.
  /**
   * Transfer from account.
   *
   * @param ctx CC context.
   * @param fromType Account type to transfer from.
   * @param from Account to transfer from.
   * @param toType Account type to transfer to.
   * @param to Account to transfer to.
   * @param amount Amount to transfer.
   * @return Donor user balance.
   */
  @Transaction
  public UserBalance transferFrom(
      final Context ctx, final String fromType, final String from,
      final String toType, final String to, final long amount,
      final boolean expirable
  ) {
    logger.info("From: " + from);
    logger.info("To: " + to);
    logger.info("Amount: " + amount);

    if (amount < 1) {
      throw new ChaincodeException("Incorrect amount");
    }

    String fromAccount = identityHelper.getUserAccount(ctx, fromType, from);
    logger.info("FromAccount: " + fromAccount);

    String toAccount = identityHelper.getUserAccount(ctx, toType, to);
    logger.info("ToAccount: " + toAccount);

    UserBalance fromBalance = getBalance(ctx, fromAccount, from);

    if (fromBalance.getBalance() < amount) {
      throw new ChaincodeException("Not enough coins");
    }

    decreaseAccountBalance(ctx, fromAccount, amount);
    if (expirable) {
      addExpirableTransaction(
          ctx, toAccount, contextHelper.getNextId(), amount, contextHelper.getCurrentTimestamp());
    } else {
      changeBalance(ctx, toAccount, amount);
    }

    return getBalance(ctx, fromAccount, from);
  }

  /**
   * Make batch transfer.
   *
   * @param ctx CC context.
   * @param transferRequestsJson Batch request JSON.
   * @return Current user balance.
   */
  @Transaction
  public UserBalance batchTransfer(
      final Context ctx, final String transferRequestsJson) {
    logger.info("Transfer requests json: " + transferRequestsJson);

    ObjectConverter objectConverter = getObjectConverter();
    List<?> args = objectConverter.deserialize(transferRequestsJson, List.class);

    if (args.size() != 2) {
      throw new ChaincodeException("Wrong amount of arguments");
    }

    Object testObject = args.get(0);
    if (!(testObject instanceof List)) {
      throw new ChaincodeException("Invalid first argument");
    }
    List<Map<String, Object>> transferRequests = (List<Map<String, Object>>) args.get(0);

    testObject = args.get(1);
    if (!(testObject instanceof Boolean)) {
      throw new ChaincodeException("Invalid second argument");
    }
    Boolean expirable = (Boolean) args.get(1);

    long total = transferRequests.stream()
        .map(r -> (long) r.get(TransferRequest.AMOUNT_KEY))
        .reduce(0L, Long::sum);

    IdentityHelper identityHelper = getIdentityHelper();
    String currentUserId = identityHelper.getCurrentUserId(ctx);
    String currentUserAccount = identityHelper
        .getUserAccount(ctx, USER_ACCOUNT_TYPE, currentUserId);
    logger.info("CurrentUserAccount: " + currentUserAccount);

    UserBalance userBalance = getBalance(ctx, currentUserAccount, currentUserId);
    logger.info("CurrentUserBalance: " + userBalance.getBalance());
    decreaseAccountBalance(ctx, currentUserAccount, total);

    for (Map<String, Object> request : transferRequests) {
      String receiverAccount = identityHelper.getUserAccount(
          ctx, USER_ACCOUNT_TYPE, String.valueOf(request.get(TransferRequest.USER_ID_KEY)));

      if (expirable) {
        addExpirableTransaction(
            ctx, receiverAccount, contextHelper.getNextId(),
            (long) request.get(TransferRequest.AMOUNT_KEY), contextHelper.getCurrentTimestamp());
      } else {
        changeBalance(ctx, receiverAccount, (long) request.get(TransferRequest.AMOUNT_KEY));
      }
    }

    return getBalance(ctx, currentUserAccount, currentUserId);
  }

  // TODO: Investigate if this method is needed and add tests.
  /**
   * Make refund.
   *
   * @param ctx CC chaincode.
   * @param projectId Project id.
   * @param receiver Receiver id.
   * @param amount Amount to refund.
   * @return Current user id.
   */
  @Transaction
  public UserBalance refund(
      final Context ctx, final String projectId, final String receiver, long amount
  ) {
    logger.info("Receiver: " + receiver);
    logger.info("Amount: " + amount);

    if (amount < 1) {
      throw new ChaincodeException("Incorrect amount");
    }

    IdentityHelper identityHelper = getIdentityHelper();
    String currentUserId = identityHelper.getCurrentUserId(ctx);

    ContextHelper contextHelper = getContextHelper();
    String minter = contextHelper.getState(ctx, MINTER_KEY, String.class);
    logger.info("Minter: " + minter);

    if (!currentUserId.equals(minter)) {
      throw new ChaincodeException("Access denied");
    }

    String projectAccount = identityHelper.getUserAccount(ctx, PROJECT_ACCOUNT_TYPE, projectId);
    logger.info("ProjectAccount: " + projectAccount);

    String receiverAccount = identityHelper.getUserAccount(ctx, USER_ACCOUNT_TYPE, receiver);
    logger.info("ReceiverAccount: " + receiverAccount);

    UserBalance projectBalance = getBalance(ctx, projectAccount, projectId);
    if (projectBalance.getBalance() < amount) {
      throw new ChaincodeException("Insufficient funds");
    }

    // TODO: Optimize this case with overloaded method with list of balance changes.
    changeBalance(ctx, projectAccount, -1 * amount);
    changeBalance(ctx, receiverAccount, amount);

    return getBalance(ctx, projectAccount, projectId);
  }

  /**
   * Make a batch refund.
   *
   * @param ctx CC context.
   * @param projectId Project id.
   * @param transferRequestsJson Request json.
   * @return User balance.
   */
  @Transaction
  public UserBalance batchRefund(
      final Context ctx, final String projectId, final String transferRequestsJson
  ) {
    logger.info("Refund requests json: " + transferRequestsJson);

    ObjectConverter objectConverter = getObjectConverter();
    TransferRequest[] transferRequests =
        objectConverter.deserialize(transferRequestsJson, TransferRequest[].class);

    long total = Stream.of(transferRequests)
        .map(TransferRequest::getAmount)
        .reduce(0, Integer::sum);

    IdentityHelper identityHelper = getIdentityHelper();
    String currentUserId = identityHelper.getCurrentUserId(ctx);

    ContextHelper contextHelper = getContextHelper();
    String minter = contextHelper.getState(ctx, MINTER_KEY, String.class);
    logger.info("Minter: " + minter);

    if (currentUserId.equals(minter)) {
      throw new ChaincodeException("Access denied");
    }

    String projectAccount = identityHelper.getUserAccount(ctx, PROJECT_ACCOUNT_TYPE, projectId);
    logger.info("ProjectAccount: " + projectAccount);

    UserBalance projectBalance = getBalance(ctx, projectAccount, projectId);
    logger.info("CurrentProjectBalance: " + projectBalance.getBalance());
    if (total != projectBalance.getBalance()) {
      throw new ChaincodeException("All money must be refunded");
    }

    changeBalance(ctx, projectAccount, -1 * total);
    for (TransferRequest request : transferRequests) {
      String receiverAccount = identityHelper
          .getUserAccount(ctx, USER_ACCOUNT_TYPE, request.getUserId());
      changeBalance(ctx, receiverAccount, request.getAmount());
    }

    return getBalance(ctx, projectAccount, projectId);
  }

  /**
   * Get balance of account.
   *
   * @param ctx CC context.
   * @param accountType Account type.
   * @param accountId Account id.
   * @return Account balance.
   */
  @Transaction(intent = TYPE.EVALUATE)
  public UserBalance balanceOf(
      final Context ctx, final String accountType, final String accountId
  ) {
    logger.info("AccountType: " + accountType);
    logger.info("AccountId: " + accountId);

    IdentityHelper identityHelper = getIdentityHelper();
    String account = identityHelper.getUserAccount(ctx, accountType, accountId);

    logger.info("Account: " + account);
    return getBalance(ctx, account, accountId);
  }

  /**
   * Get balance of batch of accounts.
   *
   * @param ctx CC context.
   * @param ids User id list.
   * @return Array of balances.
   */
  @Transaction(intent = TYPE.EVALUATE)
  public UserBalance[] batchBalanceOf(final Context ctx, final String[] ids) {
    logger.info("UserIds: " + String.join(", ", ids));

    IdentityHelper identityHelper = getIdentityHelper();
    return Stream.of(ids)
        .map(id -> getBalance(ctx, identityHelper.getUserAccount(ctx, USER_ACCOUNT_TYPE, id), id))
        .toArray(UserBalance[]::new);
  }

  /**
   * Get balance of account.
   *
   * @param ctx CC context.
   * @param account Account.
   * @param userId User identifier.
   * @return Account balance.
   */
  private UserBalance getBalance(final Context ctx, final String account, final String userId) {
    ContextHelper contextHelper = getContextHelper();
    Map<String, Long> balanceMap =
        (Map<String, Long>) contextHelper.getMap(ctx, BALANCE_MAP_KEY);

    long balance = balanceMap.getOrDefault(account, 0L);

    Map<String, List<Map<String, ?>>> expTrMap = (Map<String, List<Map<String, ?>>>)
        contextHelper.getMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY);
    logger.info("Expirable transactions map: " + expTrMap.toString());

    long expirableBalance = Optional.ofNullable(expTrMap.get(account))
        .map(list -> list.stream()
            .filter(getExpirableTransactionPredicate())
            .mapToLong(m -> (Long) m.get(ExpirableTransaction.AMOUNT_KEY))
            .reduce(0L, Long::sum))
        .orElse(0L);

    return new UserBalance(userId, balance + expirableBalance);
  }

  /**
   * Get only instance of identity helper for whole contract.
   *
   * @return IdentityHelper instance.
   */
  private IdentityHelper getIdentityHelper() {
    if (identityHelper == null) {
      identityHelper = new IdentityHelper();
    }
    return identityHelper;
  }

  /**
   * Get only instance of context helper.
   *
   * @return Context helper instance.
   */
  private ContextHelper getContextHelper() {
    if (contextHelper == null) {
      contextHelper = new ContextHelper();
    }
    return contextHelper;
  }

  /**
   * Get object converter instance.
   *
   * @return Object coverter instance.
   */
  private ObjectConverter getObjectConverter() {
    if (objectConverter == null) {
      objectConverter = new ObjectConverter();
    }
    return objectConverter;
  }

  /**
   * Add expirable transaction.
   *
   * @param ctx CC context.
   * @param account Account.
   * @param txId Transaction id.
   * @param amount Amount.
   * @param createdAt Created at timestamp.
   */
  private void addExpirableTransaction(
      final Context ctx, final String account, final String txId,
      final long amount, final long createdAt
  ) {
    ContextHelper contextHelper = getContextHelper();
    Map<String, List<Map<String, Object>>> exTrMap = (Map<String, List<Map<String, Object>>>)
        contextHelper.getMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY);

    exTrMap.putIfAbsent(account, new LinkedList<>());
    Map<String, Object> transaction = new HashMap<>();
    transaction.put(ExpirableTransaction.ID_KEY, txId);
    transaction.put(ExpirableTransaction.AMOUNT_KEY, amount);
    transaction.put(ExpirableTransaction.CREATED_AT_KEY, createdAt);
    exTrMap.get(account).add(transaction);

    contextHelper.writeMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY, exTrMap);
  }

  /**
   * Burn outdated expirable transactions.
   *
   * @param ctx CC context.
   * @param account Account.
   */
  private void flushExpiredTransactions(final Context ctx, final String account) {
    ContextHelper contextHelper = getContextHelper();
    Map<String, List<Map<String, ?>>> exTrMap = (Map<String, List<Map<String, ?>>>)
        contextHelper.getMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY);

    if (!exTrMap.containsKey(account)) {
      return;
    }

    if (exTrMap.get(account) == null || exTrMap.get(account).isEmpty()) {
      exTrMap.remove(account);
      contextHelper.writeMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY, exTrMap);
      return;
    }

    Iterator<Map<String, ?>> iterator = exTrMap.get(account).iterator();
    Predicate<Map<String, ?>> predicate = getExpirableTransactionPredicate();
    boolean mutated = false;
    while (iterator.hasNext()) {
      if (!predicate.test(iterator.next())) {
        iterator.remove();
        mutated = true;
      }
    }

    if (exTrMap.get(account).isEmpty()) {
      exTrMap.remove(account);
    }

    if (mutated) {
      contextHelper.writeMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY, exTrMap);
    }
  }

  /**
   * Decrease account balance including permanent balance and expirable transactions.
   *
   * @param ctx CC context.
   * @param account Account.
   * @param amount Amount to decrease.
   */
  private void decreaseAccountBalance(final Context ctx, final String account, long amount) {
    UserBalance userBalance = getBalance(ctx, account, account);
    if (amount < 1) {
      return;
    }
    if (userBalance.getBalance() < amount) {
      throw new ChaincodeException("Balance amount is less than needed");
    }

    ContextHelper contextHelper = getContextHelper();
    Map<String, List<Map<String, Object>>> exTrMap = (Map<String, List<Map<String, Object>>>)
        contextHelper.getMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY);

    List<Map<String, Object>> expirableTransactions = exTrMap.get(account);
    if (expirableTransactions != null && !expirableTransactions.isEmpty()) {
      Iterator<Map<String, Object>> iterator = expirableTransactions.iterator();
      Predicate<Map<String, ?>> predicate = getExpirableTransactionPredicate();

      while (iterator.hasNext() && amount > 0) {
        Map<String, Object> transaction = iterator.next();
        if (!predicate.test(transaction)) {
          iterator.remove();
        } else {
          Long trAmount = (Long) transaction.get(ExpirableTransaction.AMOUNT_KEY);
          if (amount >= trAmount) {
            amount -= trAmount;
            iterator.remove();
          } else {
            transaction.put(ExpirableTransaction.AMOUNT_KEY, trAmount - amount);
            amount = 0;
          }
        }
      }

      if (exTrMap.get(account).isEmpty()) {
        exTrMap.remove(account);
      }

      contextHelper.writeMap(ctx, EXPIRABLE_TRANSACTION_MAP_KEY, exTrMap);
    }

    if (amount > 0) {
      changeBalance(ctx, account, -1 * amount);
    }
  }

  /**
   * Change permanent balance of account.
   *
   * @param ctx CC context.
   * @param account Account.
   * @param amount Amount to change.
   */
  private void changeBalance(final Context ctx, final String account, final long amount) {
    ContextHelper contextHelper = getContextHelper();
    Map<String, Long> balanceMap = (Map<String, Long>)
        contextHelper.getMap(ctx, BALANCE_MAP_KEY);

    balanceMap.putIfAbsent(account, 0L);
    long rest = balanceMap.get(account) + amount;
    if (rest < 0) {
      throw new ChaincodeException("Insufficient funds on '" + account + "' account");
    }

    balanceMap.put(account, rest);
    contextHelper.writeMap(ctx, BALANCE_MAP_KEY, balanceMap);
  }

  /**
   * Get predicate to check whether an expiration transaction is expired.
   *
   * @return Predicate for checking transaction expiration.
   */
  private Predicate<Map<String, ?>> getExpirableTransactionPredicate() {
    long threshold = getContextHelper().getCurrentTimestamp() - TX_EXPIRATION_PERIOD;
    return t -> ((Long) t.get(ExpirableTransaction.CREATED_AT_KEY)) > threshold;
  }
}
