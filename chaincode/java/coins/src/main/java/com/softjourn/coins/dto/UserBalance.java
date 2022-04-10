package com.softjourn.coins.dto;

import com.owlike.genson.annotation.JsonProperty;
import java.util.Objects;
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

  @Override
  public String toString() {
    return "{userId: '" + userId + "', balance: " + balance +"}";
  }

  @Override
  public int hashCode() {
    int hash = 3;
    hash = 53 * hash + (this.userId != null ? this.userId.hashCode() : 0);
    hash = 53 * hash + ((int) this.balance % Integer.MAX_VALUE);
    return hash;
  }

  @Override
  public boolean equals(Object obj) {
    if (obj == null) {
      return false;
    }

    if (obj.getClass() != this.getClass()) {
      return false;
    }

    final UserBalance other = (UserBalance) obj;
    return Objects.equals(this.userId, other.userId) && Objects.equals(this.balance, other.balance);
  }
}
