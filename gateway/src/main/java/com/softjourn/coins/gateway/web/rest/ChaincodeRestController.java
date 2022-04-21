package com.softjourn.coins.gateway.web.rest;

import com.softjourn.coins.gateway.dto.ChaincodeRequestDto;
import com.softjourn.coins.gateway.service.ChaincodeService;
import com.softjourn.coins.gateway.service.JwtService;
import java.util.Map;
import java.util.UUID;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@Slf4j
@RestController
@RequiredArgsConstructor
public class ChaincodeRestController {

  private final ChaincodeService chaincodeService;
  private final JwtService jwtService;

  @PostMapping({"/invoke", "/invoke/{chaincode}"})
  public ResponseEntity<Map<String, Object>> invoke(
      @PathVariable(required = false) String chaincode,
      @RequestBody ChaincodeRequestDto chaincodeRequest,
      JwtAuthenticationToken principal
  ) {
    String requestId = generateRequestId();
    log.info("Invoke request #{}: {}", requestId, chaincodeRequest.toString());
    ResponseEntity<Map<String, Object>> response = ResponseEntity.ok(
        chaincodeService.invoke(jwtService.getUsername(principal), chaincode, chaincodeRequest));
    log.info("Invoke response #{}: {}", requestId, response.toString());
    return response;
  }

  @PostMapping({"/query", "/query/{chaincode}"})
  public ResponseEntity<Map<String, Object>> query(
      @PathVariable(required = false) String chaincode,
      @RequestBody ChaincodeRequestDto chaincodeRequest,
      JwtAuthenticationToken principal
  ) {
    String requestId = generateRequestId();
    log.info("Query request #{}: {}", requestId, chaincodeRequest.toString());
    ResponseEntity<Map<String, Object>> response = ResponseEntity.ok(
        chaincodeService.query(jwtService.getUsername(principal), chaincode, chaincodeRequest));
    log.info("Query response #{}: {}", requestId, response.toString());
    return response;
  }

  private String generateRequestId() {
    return UUID.randomUUID().toString();
  }
}
