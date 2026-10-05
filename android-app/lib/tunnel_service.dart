import 'package:flutter/services.dart';

class TunnelService {
  static const _ch = MethodChannel('zarvand/tunnel');
  static const _events = EventChannel('zarvand/tunnel_state');
  static Stream<String> get stateStream => _events.receiveBroadcastStream().cast<String>();

  static Future<void> start({
    required List<String> domains,
    required String key,
    required String server,
  }) =>
      _ch.invokeMethod('start', {
        'domains': domains,
        'key': key,
        'server': server,
      });

  static Future<void> stop() => _ch.invokeMethod('stop');

  static List<String> _logs = [];
  static List<String> drainLogs() {
    final l = _logs;
    _logs = [];
    return l;
  }

  static void pushLog(String line) => _logs.add(line);
}
