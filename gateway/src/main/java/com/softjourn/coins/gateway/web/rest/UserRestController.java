package com.softjourn.coins.gateway.web.rest;

import com.softjourn.coins.gateway.dto.EnrollmentRequestDto;
import com.softjourn.coins.gateway.service.AdminService;
import com.softjourn.coins.gateway.service.JwtService;
import com.softjourn.coins.gateway.service.UserService;
import java.util.HashMap;
import java.util.Map;
import lombok.RequiredArgsConstructor;
import org.hyperledger.fabric_ca.sdk.exception.EnrollmentException;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequiredArgsConstructor
public class UserRestController {

  private final JwtService jwtService;
  private final AdminService adminService;
  private final UserService userService;

  @PostMapping("/enroll")
  public ResponseEntity<Map<String, String>> enroll(
      @RequestBody EnrollmentRequestDto requestDto
  ) throws EnrollmentException {
    adminService.enroll();
    userService.enroll(requestDto.getUsername());
    return ResponseEntity.ok(new HashMap<>() {{
      put("token", jwtService.generateToken(requestDto.getUsername(), requestDto.getOrgName()));
    }});
  }
}
