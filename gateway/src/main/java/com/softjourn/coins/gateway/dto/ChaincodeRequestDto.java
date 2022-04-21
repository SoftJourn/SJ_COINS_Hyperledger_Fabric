package com.softjourn.coins.gateway.dto;

import java.util.Arrays;
import lombok.Data;

@Data
public class ChaincodeRequestDto {

  private Boolean isObject;
  private String fcn;
  private String[] args;

  @Override
  public String toString() {
    return String.format("{isObject: %b, fcn: %s, args: %s}", isObject, fcn, Arrays.asList(args));
  }
}
