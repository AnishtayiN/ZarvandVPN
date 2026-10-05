import 'package:flutter/services.dart';

class Settings {
  List<String> domains;
  String key;
  int encryptionMethod; // 0..5
  String protocol;      // SOCKS5 | TCP
  String listenIp;
  int listenPort;
  bool localDns;
  String localDnsIp;
  int localDnsPort;
  List<String> resolvers; // DNS resolver IPs
  int packetDuplication;
  int uploadCompression;   // 0 none,1 gzip,2 zstd,3 lz4
  int downloadCompression;
  String logLevel;         // DEBUG INFO WARN ERROR
  int rxTxWorkers;

  Settings({
    this.domains = const [],
    this.key = '',
    this.encryptionMethod = 1,
    this.protocol = 'SOCKS5',
    this.listenIp = '127.0.0.1',
    this.listenPort = 18000,
    this.localDns = false,
    this.localDnsIp = '127.0.0.1',
    this.localDnsPort = 53,
    this.resolvers = const ['8.8.8.8', '1.1.1.1'],
    this.packetDuplication = 3,
    this.uploadCompression = 0,
    this.downloadCompression = 0,
    this.logLevel = 'INFO',
    this.rxTxWorkers = 4,
  });

  Map<String, dynamic> toMap() => {
        'domains': domains,
        'key': key,
        'encryptionMethod': encryptionMethod,
        'protocol': protocol,
        'listenIp': listenIp,
        'listenPort': listenPort,
        'localDns': localDns,
        'localDnsIp': localDnsIp,
        'localDnsPort': localDnsPort,
        'resolvers': resolvers,
        'packetDuplication': packetDuplication,
        'uploadCompression': uploadCompression,
        'downloadCompression': downloadCompression,
        'logLevel': logLevel,
        'rxTxWorkers': rxTxWorkers,
      };

  static Settings fromMap(Map m) => Settings(
        domains: (m['domains'] as List?)?.cast<String>() ?? const [],
        key: m['key'] ?? '',
        encryptionMethod: m['encryptionMethod'] ?? 1,
        protocol: m['protocol'] ?? 'SOCKS5',
        listenIp: m['listenIp'] ?? '127.0.0.1',
        listenPort: m['listenPort'] ?? 18000,
        localDns: m['localDns'] ?? false,
        localDnsIp: m['localDnsIp'] ?? '127.0.0.1',
        localDnsPort: m['localDnsPort'] ?? 53,
        resolvers: (m['resolvers'] as List?)?.cast<String>() ?? const ['8.8.8.8'],
        packetDuplication: m['packetDuplication'] ?? 3,
        uploadCompression: m['uploadCompression'] ?? 0,
        downloadCompression: m['downloadCompression'] ?? 0,
        logLevel: m['logLevel'] ?? 'INFO',
        rxTxWorkers: m['rxTxWorkers'] ?? 4,
      );
}

class TunnelService {
  static const _ch = MethodChannel('arefdns/tunnel');
  static const _events = EventChannel('arefdns/tunnel_state');
  static Stream<String> get stateStream =>
      _events.receiveBroadcastStream().cast<String>();

  static Future<void> start(Settings s) =>
      _ch.invokeMethod('start', s.toMap());

  static Future<void> stop() => _ch.invokeMethod('stop');

  static Future<bool> hasPermission() async =>
      await _ch.invokeMethod('vpnPermission') == true;

  static Future<void> requestPermission() => _ch.invokeMethod('requestVpnPermission');

  static List<String> _logs = [];
  static List<String> drainLogs() {
    final l = _logs;
    _logs = [];
    return l;
  }

  static void pushLog(String line) => _logs.add(line);
}
