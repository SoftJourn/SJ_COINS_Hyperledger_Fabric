package com.softjourn.coins.dto;

import com.owlike.genson.annotation.JsonProperty;
import org.hyperledger.fabric.contract.annotation.DataType;
import org.hyperledger.fabric.contract.annotation.Property;

@DataType
public class UserBalance {

  @Property
  private final String userId;

  @Property
  private final long balance;

  public UserBalance(@JsonProperty("userId") String userId, @JsonProperty("balance") long balance) {
    this.userId = userId;
    this.balance = balance;
  }

  public String getUserId() {
    return userId;
  }

  public long getBalance() {
    return balance;
  }
}
