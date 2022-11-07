package com.softjourn.coins;

import com.softjourn.coins.dto.UserBalance;
import com.softjourn.common.helper.ContextHelper;
import com.softjourn.common.helper.IdentityHelper;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.stream.Stream;
import org.hyperledger.fabric.contract.Context;
import org.hyperledger.fabric.shim.ChaincodeException;
import org.hyperledger.fabric.shim.ChaincodeStub;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;
import org.mockito.Matchers;
import org.mockito.Mockito;

public class SmartContractTest {

  private ContextHelper contextHelper;
  private IdentityHelper identityHelper;
  private Context context;
  private ChaincodeStub stub;
  private SmartContract target;

  @BeforeEach
  public void before() {
    contextHelper = Mockito.mock(ContextHelper.class);
    identityHelper = Mockito.mock(IdentityHelper.class);
    context = Mockito.mock(Context.class);
    stub = Mockito.mock(ChaincodeStub.class);
    target = new SmartContract(contextHelper, identityHelper);

    Mockito.when(context.getStub()).thenReturn(stub);
  }


  // Init ledger method.
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

    Assertions.assertEquals(currencyName, target.initLedger(context));

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
        ChaincodeException.class, () -> target.initLedger(context),
        "Expected initLedger to throw, but it didn't"
    );

    Assertions.assertEquals(
        "Incorrect number of arguments. Expected 2, was 1", thrownException.getMessage());
  }


  // Mint method.
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
      long amount, long start, String minter, Map<String, Long> balanceMap
  ) {
    Mockito.when(contextHelper.getState(context, "minter", String.class))
        .thenReturn(minter);
    Mockito.when(identityHelper.getCurrentUserId(context)).thenReturn(minter);
    Mockito.when(identityHelper.getUserAccount(context, "user_", minter))
        .thenReturn("user_" + minter);
    Mockito.when(contextHelper.<String, Long>getMap(context, "balances"))
        .thenReturn((Map) balanceMap);

    UserBalance userBalance = target.mint(context, amount);
    Assertions.assertEquals(new UserBalance(minter, start + amount), userBalance);

    balanceMap.put("user_" + minter, balanceMap.get("user_" + minter) + amount);
    Mockito.verify(contextHelper).writeMap(context, "balances", balanceMap);
  }


  // Transfer method.
  public static Stream<Object[]> transfer_correctParameters_success() {
    return Stream.of(
        new Object[]{
            "user_", "receiver", 100L, false,
            "sj_coin", 0L,
            new HashMap<>() {{ put("user_sj_coin", 100L); }},
            new HashMap<>() {{
              put("user_sj_coin", 0L);
              put("user_receiver", 100L);
            }},
            new HashMap<>(),
            new HashMap<>()
        },
        new Object[]{
            "user_", "receiver", 120L, false,
            "sj_coin", 80L,
            new HashMap<>() {{
              put("user_sj_coin", 200L);
              put("user_unknown", 10L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 80L);
              put("user_unknown", 10L);
              put("user_receiver", 120L);
            }},
            new HashMap<>(),
            new HashMap<>()
        },
        new Object[]{
            "user_", "receiver", 120L, false,
            "sj_coin", 80L,
            new HashMap<>() {{
              put("user_sj_coin", 200L);
              put("user_unknown", 10L);
              put("user_receiver", 62L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 80L);
              put("user_unknown", 10L);
              put("user_receiver", 182L);
            }},
            new HashMap<>(),
            new HashMap<>()
        },
        new Object[]{
            "user_", "receiver", 10L, false,
            "sj_coin", 178L,
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_unknown", 10L);
              put("user_receiver", 62L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_unknown", 10L);
              put("user_receiver", 72L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 88L);
                  put("createdAt", 13500L);
                }});
              }});
            }},
            new HashMap<>() {{
              put("user_sj_coin", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 78L);
                  put("createdAt", 13500L);
                }});
              }});
            }}
        },
        new Object[]{
            "user_", "receiver", 10L, false,
            "sj_coin", 95L,
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 95L);
              put("user_receiver", 210L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 5L);
                  put("createdAt", 13500L);
                }});
              }});
            }},
            new HashMap<>()
        },
        new Object[]{
            "user_", "receiver", 10L, true,
            "sj_coin", 90L,
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 90L);
              put("user_receiver", 200L);
            }},
            new HashMap<>(),
            new HashMap<>() {{
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 10L);
                  put("createdAt", 13600L);
                }});
              }});
            }}
        },
        new Object[]{
            "user_", "receiver", 10L, true,
            "sj_coin", 90L,
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 90L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 13500L);
                }});
              }});
            }},
            new HashMap<>() {{
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 13500L);
                }});
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 10L);
                  put("createdAt", 13600L);
                }});
              }});
            }}
        },
        new Object[]{
            "user_", "receiver", 10L, true,
            "sj_coin", 91L,
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 91L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 13500L);
                }});
              }});
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 13500L);
                }});
              }});
            }},
            new HashMap<>() {{
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 13500L);
                }});
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 10L);
                  put("createdAt", 13600L);
                }});
              }});
            }}
        },
        new Object[]{
            "user_", "receiver", 10L, true,
            "sj_coin", 90L,
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 90L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 1000L);
                }});
              }});
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 13500L);
                }});
              }});
            }},
            new HashMap<>() {{
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 13500L);
                }});
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 10L);
                  put("createdAt", 13600L);
                }});
              }});
            }}
        },
        new Object[]{
            "user_", "receiver", 10L, true,
            "sj_coin", 90L,
            new HashMap<>() {{
              put("user_sj_coin", 100L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", 90L);
              put("user_receiver", 200L);
            }},
            new HashMap<>() {{
              put("user_sj_coin", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 1000L);
                }});
              }});
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "0");
                  put("amount", 1L);
                  put("createdAt", 1000L);
                }});
              }});
            }},
            new HashMap<>() {{
              put("user_receiver", new LinkedList<>() {{
                add(new HashMap<>() {{
                  put("id", "1");
                  put("amount", 10L);
                  put("createdAt", 13600L);
                }});
              }});
            }}
        }
    );
  }

  @ParameterizedTest
  @MethodSource
  public void transfer_correctParameters_success(
      String receiverAccountType, String receiver, long amount, boolean expirable,
      String senderId, long rest,
      Map<String, Long> balanceMap, Map<String, Long> expectedBalanceMap,
      Map<String, List<Map<String, Object>>> expTrMap,
      Map<String, List<Map<String, Object>>> expectedExpTrMap
  ) {
    ContextHelperImpl contextHelper = new ContextHelperImpl();
    contextHelper.putState("balances", balanceMap);
    contextHelper.putState("expirable_transactions", expTrMap);
    contextHelper.setNextId("1");
    contextHelper.setCurrentTimestamp(13600L);
    target = new SmartContract(contextHelper, identityHelper);

    Mockito.when(identityHelper.getCurrentUserId(context)).thenReturn(senderId);
    Mockito.when(identityHelper
        .getUserAccount(Matchers.eq(context), Matchers.anyString(), Matchers.anyString()))
        .then(i -> i.getArgumentAt(1, String.class) + i.getArgumentAt(2, String.class));

    UserBalance result = target.transfer(context, receiverAccountType, receiver, amount, expirable);

    Assertions.assertEquals(new UserBalance(senderId, rest), result);
    Assertions.assertEquals(
        expectedBalanceMap, contextHelper.getState(context,"balances", Map.class));
    Assertions.assertEquals(
        expectedExpTrMap, contextHelper.getState(context,"expirable_transactions", Map.class));
  }


  // TransferFrom method.
  public static Stream<Object[]> transferFrom_correctParameters_success() {
    return Stream.of(
        new Object[]{
            "user_", "donor",
            "user_", "recipient", 100L, false, 134L,

            new HashMap<>() {{ put("user_donor", 234L); put("user_recipient", 89L); }},
            new HashMap<>(),
            new HashMap<>() {{ put("user_donor", 134L); put("user_recipient", 189L); }},
            new HashMap<>()
        },
        new Object[]{
            "user_", "donor",
            "user_", "recipient", 12L, true, 222L,

            new HashMap<>() {{ put("user_donor", 234L); put("user_recipient", 89L); }},
            new HashMap<>(),
            new HashMap<>() {{ put("user_donor", 222L); put("user_recipient", 89L); }},
            new HashMap<>() {{
              put("user_recipient", new ArrayList<Map<String, Object>>() {{
                add(new HashMap<>() {{ put("createdAt", 13600L); put("amount", 12L); put("id", "1"); }});
              }});
            }}
        }
    );
  }

  @ParameterizedTest
  @MethodSource
  public void transferFrom_correctParameters_success(
      String donorAccountType, String donor,
      String recipientAccountType, String recipient,
      long amount, boolean expirable, long left,

      Map<String, Long> initialBalanceMap,
      Map<String, List<Map<String, Object>>> initialExpirableTransactionsMap,
      Map<String, Long> expectedBalanceMap,
      Map<String, List<Map<String, Object>>> expectedExpirableTransactionsMap
  ) {
    ContextHelperImpl contextHelper = new ContextHelperImpl();
    contextHelper.putState("balances", initialBalanceMap);
    contextHelper.putState("expirable_transactions", initialExpirableTransactionsMap);
    contextHelper.setNextId("1");
    contextHelper.setCurrentTimestamp(13600L);
    target = new SmartContract(contextHelper, identityHelper);

    Mockito.when(identityHelper
        .getUserAccount(Matchers.eq(context), Matchers.anyString(), Matchers.anyString()))
        .then(i -> i.getArgumentAt(1, String.class) + i.getArgumentAt(2, String.class));

    UserBalance result = target.transferFrom(
        context, donorAccountType, donor, recipientAccountType, recipient, amount, expirable);

    Assertions.assertEquals(new UserBalance(donor, left), result);
    Assertions.assertEquals(
        expectedBalanceMap, contextHelper.getState(context,"balances", Map.class));
    Assertions.assertEquals(
        expectedExpirableTransactionsMap, contextHelper.getState(context,"expirable_transactions", Map.class));
  }
}
