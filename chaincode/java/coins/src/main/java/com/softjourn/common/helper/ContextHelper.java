package com.softjourn.common.helper;

import java.lang.reflect.InvocationTargetException;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import org.hyperledger.fabric.contract.Context;
import org.hyperledger.fabric.shim.ChaincodeException;

public class ContextHelper {

  private final ObjectConverter converter = new ObjectConverter();
  private final Map<String, Object> writeStateCache = new HashMap<>();

  /**
   * Get state value from context. Consider write state cache map to read updated state version.
   *
   * @param ctx Chaincode context.
   * @param key String-value key of state value.
   * @param <K> Parameter for key.
   * @param <V> Parameter for value.
   * @return State value in form of Map structure.
   */
  public <K, V> Map<? extends K, ? extends V> getMap(final Context ctx, final String key) {
    if (writeStateCache.containsKey(key)) {
      return (Map<K, V>) writeStateCache.get(key);
    }

    byte[] content = ctx.getStub().getState(key);
    if (content.length < 1) {
      return new HashMap<>();
    }

    return converter.deserialize(content, Map.class);
  }

  /**
   * Write map to chaincode context. And populating write state cache.
   * @param ctx Chaincode context.
   * @param key String-value key of state value.
   * @param value Value object of state.
   */
  public void writeMap(final Context ctx, final String key, final Map<?, ?> value) {
    writeStateCache.put(key, value);
    ctx.getStub().putState(key, converter.serialize(value));
  }

  /**
   * Get state by parametrization.
   *
   * @param ctx Context.
   * @param key Key.
   * @param clazz Class of instance to be returned.
   * @param <T> Type parameter.
   * @return Instance of required type.
   */
  public <T> T getState(final Context ctx, final String key, Class<T> clazz) {
    if (writeStateCache.containsKey(key)) {
      return (T) writeStateCache.get(key);
    }

    byte[] content = ctx.getStub().getState(key);
    if (content.length < 1) {
      try {
        return (T) Arrays.stream(clazz.getConstructors())
            .filter(c -> c.getParameterCount() == 0)
            .findAny()
            .orElseThrow(() -> new ChaincodeException("Appropriate constructor is absent"))
            .newInstance();
      } catch (InstantiationException | IllegalAccessException | InvocationTargetException e) {
        throw new ChaincodeException(e);
      }
    }

    return (T) converter.deserialize(content, clazz);
  }
}
