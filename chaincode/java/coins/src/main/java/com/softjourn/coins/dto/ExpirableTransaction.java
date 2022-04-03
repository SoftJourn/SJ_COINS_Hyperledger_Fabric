package com.softjourn.coins.dto;

import com.owlike.genson.annotation.JsonProperty;
import org.hyperledger.fabric.contract.annotation.DataType;
import org.hyperledger.fabric.contract.annotation.Property;

@DataType
public class ExpirableTransaction {

  public static final String ID_KEY = "id";
  public static final String AMOUNT_KEY = "amount";
  public static final String CREATED_AT_KEY = "createdAt";

  @Property
  private long amount;

  @Property
  private final String id;

  @Property
  private final long createdAt;

  public ExpirableTransaction(
      @JsonProperty("amount") long amount,
      @JsonProperty("id") String id,
      @JsonProperty("createdAt") long createdAt
  ) {
    this.amount = amount;
    this.id = id;
    this.createdAt = createdAt;
  }

  public long getAmount() {
    return amount;
  }

  public String getId() {
    return id;
  }

  public long getCreatedAt() {
    return createdAt;
  }

  public void setAmount(long amount) {
    this.amount = amount;
  }
}
