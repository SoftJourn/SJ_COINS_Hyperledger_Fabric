package com.softjourn.coins.gateway.service;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.softjourn.coins.gateway.config.ApplicationProperties;
import com.softjourn.coins.gateway.dto.ChaincodeRequestDto;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeoutException;
import java.util.function.BiFunction;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.ContractException;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;
import org.hyperledger.fabric.gateway.Wallet;
import org.springframework.stereotype.Service;

@Slf4j
@Service
@RequiredArgsConstructor
public class ChaincodeService {

  private final ApplicationProperties applicationProperties;
  private final Wallet wallet;
  private final ObjectMapper objectMapper = new ObjectMapper();

  public Map<String, Object> invoke(
      String username, String chaincode, ChaincodeRequestDto request
  ) {
    Map<String, Object> result = performInvocation(
        username,
        chaincode,
        request,
        (tx, args) -> {
          try {
            return tx.submit(args);
          } catch (ContractException | InterruptedException | TimeoutException exception) {
            throw new RuntimeException(exception);
          }
        });
    log.info("Result: {}", result);
    return result;
  }

  public Map<String, Object> query(
      String username, String chaincode, ChaincodeRequestDto request
  ) {
    Map<String, Object> result = performInvocation(
        username,
        chaincode,
        request,
        (tx, args) -> {
          try {
            return tx.evaluate(args);
          } catch (ContractException exception) {
            throw new RuntimeException(exception);
          }
        });
    log.info("Result: {}", result);
    return result;
  }

  private Map<String, Object> performInvocation(
      String username, String chaincode, ChaincodeRequestDto request,
      BiFunction<Transaction, String[], byte[]> handler
  ) {
    Gateway.Builder builder = null;
    try {
      builder = getGatewayBuilder(username);
    } catch (IOException exception) {
      throw new RuntimeException(exception);
    }

    // Create a gateway connection
    try (Gateway gateway = builder.connect()) {
      // Obtain a smart contract deployed on the network.
      Contract contract = getContract(gateway, chaincode);
      Transaction tx = contract.createTransaction(request.getFcn());
      byte[] result = null;
      if (Boolean.TRUE.equals(request.getIsObject())) {
        String[] args = new String[]{objectMapper.writeValueAsString(request.getArgs())};
        result = handler.apply(tx, args);
      } else {
        result = handler.apply(tx, request.getArgs());
      }

      return packSuccessResponse(tx.getTransactionId(), new String(result, StandardCharsets.UTF_8));
    } catch (Exception exception) {
      return packErrorResponse(exception.getMessage());
    }
  }

  private Map<String, Object> packSuccessResponse(
      String txId, String response
  ) throws JsonProcessingException {
    return new HashMap<>() {{
      put("success", true);
      put("transactionID", txId);
      put("payload", objectMapper.readValue(response, Object.class));
    }};
  }

  private Map<String, Object> packErrorResponse(String message) {
    return new HashMap<>() {{
      put("success", false);
      put("message", message);
    }};
  }

  private Contract getContract(Gateway gateway, String chaincode) {
    chaincode = getNormalizedChaincodeName(chaincode);
    Network network = gateway.getNetwork("mychannel");
    return network.getContract(chaincode);
  }

  private Gateway.Builder getGatewayBuilder(
      String username
  ) throws IOException {
    return Gateway.createBuilder()
        .identity(wallet, username)
        .networkConfig(getNetworkConfigFilePath());
  }

  private Path getNetworkConfigFilePath() {
    return Paths.get("connection.json");
  }

  private boolean isChaincodeSupported(String chaincode) {
    return applicationProperties.getSupportedChaincodes().contains(chaincode);
  }

  private String getNormalizedChaincodeName(String chaincode) {
    if (chaincode == null) {
      chaincode = applicationProperties.getDefaultChaincode();
    }

    if (!isChaincodeSupported(chaincode)) {
      throw new UnsupportedOperationException(
          String.format("Chaincode '%s' is not supported", chaincode));
    }

    return chaincode;
  }
}
