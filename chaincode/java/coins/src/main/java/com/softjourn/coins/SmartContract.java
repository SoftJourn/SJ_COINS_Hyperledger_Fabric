package com.softjourn.coins;

import com.owlike.genson.Genson;
import com.softjourn.common.helper.IdentityHelper;
import java.util.List;
import org.hyperledger.fabric.Logger;
import org.hyperledger.fabric.contract.Context;
import org.hyperledger.fabric.contract.ContractInterface;
import org.hyperledger.fabric.contract.annotation.Contact;
import org.hyperledger.fabric.contract.annotation.Contract;
import org.hyperledger.fabric.contract.annotation.Default;
import org.hyperledger.fabric.contract.annotation.Info;
import org.hyperledger.fabric.contract.annotation.License;
import org.hyperledger.fabric.contract.annotation.Transaction;
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

  private final Genson genson = new Genson();
  private final Logger logger = Logger.getLogger(getClass());

  /**
   * Init ledger with some start values.
   *
   * @param ctx The transaction context.
   */
  @Transaction(intent = Transaction.TYPE.SUBMIT)
  public String InitLedger(final Context ctx) {
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

    IdentityHelper identityHelper = new IdentityHelper();
    String currentUserId = identityHelper.getCurrentUserId(ctx);
    String currentUserAccount = identityHelper
        .getUserAccount(ctx, USER_ACCOUNT_TYPE, currentUserId);
    logger.info("CurrentUserAccount: " + currentUserAccount);

    return currencyName;
  }
}
