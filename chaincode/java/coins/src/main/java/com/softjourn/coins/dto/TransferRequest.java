package com.softjourn.coins.dto;

import com.owlike.genson.annotation.JsonProperty;
import org.hyperledger.fabric.contract.annotation.DataType;
import org.hyperledger.fabric.contract.annotation.Property;

@DataType
public class TransferRequest {

  public static final String AMOUNT_KEY = "amount";
  public static final String USER_ID_KEY = "userId";

  @Property
  private final String userId;

  @Property
  private final int amount;

  public TransferRequest(
      @JsonProperty("userId") String userId, @JsonProperty("amount") int amount
  ) {
    this.userId = userId;
    this.amount = amount;
  }

  public String getUserId() {
    return userId;
  }

  public int getAmount() {
    return amount;
  }
}
