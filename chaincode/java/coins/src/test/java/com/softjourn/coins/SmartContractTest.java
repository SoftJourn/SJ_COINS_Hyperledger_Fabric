package com.softjourn.coins;

import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.stream.Stream;
import org.hyperledger.fabric.contract.Context;
import org.hyperledger.fabric.shim.ChaincodeException;
import org.hyperledger.fabric.shim.ChaincodeStub;
import org.junit.Assert;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;
import org.mockito.Matchers;
import org.mockito.Mockito;
import com.softjourn.common.helper.*;
import com.softjourn.coins.dto.*;

public class SmartContractTest {

  private ContextHelper contextHelper;
  private IdentityHelper identityHelper;
  private Context context;
  private ChaincodeStub stub;

  @BeforeEach
  public void before() {
    contextHelper = Mockito.mock(ContextHelper.class);
    identityHelper = Mockito.mock(IdentityHelper.class);
    context = Mockito.mock(Context.class);
    stub = Mockito.mock(ChaincodeStub.class);

    Mockito.when(context.getStub()).thenReturn(stub);
  }

  static Stream<Object[]> initLedger_correctParameters_success() {
    return Stream.of(
        new Object[]{"sj_coin", "SJCoin", false},
        new Object[]{"not_sj_coin", "some-currency", false},
        new Object[]{"sj_coin", "SJCoin", true},
        new Object[]{"not_sj_coin", "some-currency", true});
  }

  @ParameterizedTest
  @MethodSource
  public void initLedger_correctParameters_success(
      String minterName, String currencyName, boolean isStateInited
  ) {
    List<String> parameters = new LinkedList<>();
    parameters.add(minterName);
    parameters.add(currencyName);

    Mockito.when(stub.getParameters()).thenReturn(parameters);
    Mockito.when(identityHelper.getCurrentUserId(Matchers.any())).thenReturn(minterName);
    Mockito.when(identityHelper.getUserAccount(context, "user_", minterName))
        .thenReturn("user_" + minterName);
    Mockito.when(contextHelper.getMap(Matchers.same(context), Matchers.anyString()))
        .thenReturn(isStateInited ? new HashMap<>() : null);

    Assert.assertEquals(
        currencyName, new SmartContract(contextHelper, identityHelper).initLedger(context));

    Mockito.verify(stub).putState("currency", currencyName.getBytes());
    Mockito.verify(stub).putState("minter", minterName.getBytes());

    Mockito.verify(contextHelper).getMap(context, "balances");
    if (!isStateInited) {
      Mockito.verify(contextHelper)
          .writeMap(Matchers.same(context), Matchers.eq("balances"), Matchers.anyMap());
    }

    Mockito.verify(contextHelper).getMap(context, "expirable_transactions");
    if (!isStateInited) {
      Mockito.verify(contextHelper).writeMap(
          Matchers.same(context), Matchers.eq("expirable_transactions"), Matchers.anyMap());
    }
  }

  @Test
  public void initLedger_incorrectParameters_fail() {
    Mockito.when(context.getStub()).thenReturn(stub);

    List<String> parameters = new LinkedList<>();
    parameters.add("sj_coin");

    Mockito.when(stub.getParameters()).thenReturn(parameters);

    Exception thrownException = Assertions.assertThrows(
        ChaincodeException.class,
        () -> new SmartContract(contextHelper, identityHelper).initLedger(context),
        "Expected doThing() to throw, but it didn't"
    );

    Assertions.assertEquals(
        "Incorrect number of arguments. Expected 2, was 1", thrownException.getMessage());
  }

  public static Stream<Object[]> mint_correctParameters_success() {
    return Stream.of(
        new Object[]{1L, 0L, "sj_coin", new HashMap<>() {{ put("user_sj_coin", 0L); }}},
        new Object[]{10_000_000L, 0L, "minter", new HashMap<>() {{ put("user_minter", 0L); }}},
        new Object[]{10_000_000L, 102L, "minter", new HashMap<>() {{ put("user_minter", 102L); }}},
        new Object[]{
            10_000_000L,
            102L,
            "minter",
            new HashMap<>() {{
              put("user_minter", 102L);
              put("user_other", 102L);
            }}}
    );
  }

  @ParameterizedTest
  @MethodSource
  public void mint_correctParameters_success(
      long amount, long start, String minter, Map balanceMap
  ) {
    Mockito.when(contextHelper.getState(context, "minter", String.class))
        .thenReturn(minter);
    Mockito.when(identityHelper.getCurrentUserId(context)).thenReturn(minter);
    Mockito.when(identityHelper.getUserAccount(context, "user_", minter))
        .thenReturn("user_" + minter);
    Mockito.when(contextHelper.<String, Object>getMap(context, "balances"))
        .thenReturn(balanceMap);

    UserBalance userBalance = new SmartContract(contextHelper, identityHelper).mint(context, amount);
    Assertions.assertEquals(new UserBalance(minter, start + amount), userBalance);

    balanceMap.put("user_" + minter, (Long) balanceMap.get("user_" + minter) + amount);
    Mockito.verify(contextHelper).writeMap(context, "balances", balanceMap);
  }
}
