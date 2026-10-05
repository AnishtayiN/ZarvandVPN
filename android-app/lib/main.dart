import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:convert';
import 'tunnel_service.dart';

void main() => runApp(const ZarvandApp());

class ZarvandApp extends StatelessWidget {
  const ZarvandApp({super.key});
  @override
  Widget build(BuildContext context) {
    final seed = const Color(0xFF2E6BFF);
    return MaterialApp(
      title: 'ZarvandVPN',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: seed),
      ),
      darkTheme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
            seedColor: seed, brightness: Brightness.dark),
      ),
      themeMode: ThemeMode.dark,
      home: const HomePage(),
    );
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> with SingleTickerProviderStateMixin {
  late TabController _tabs;
  Settings s = Settings();
  bool connected = false;
  bool connecting = false;
  final List<String> logs = [];
  bool _loaded = false;

  // controllers
  final domainsCtrl = TextEditingController();
  final keyCtrl = TextEditingController();
  final listenIpCtrl = TextEditingController();
  final listenPortCtrl = TextEditingController();
  final localDnsIpCtrl = TextEditingController();
  final localDnsPortCtrl = TextEditingController();
  final resolversCtrl = TextEditingController();
  final dupCtrl = TextEditingController();
  final workersCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 3, vsync: this);
    TunnelService.stateStream.listen((e) {
      if (!mounted) return;
      if (e == 'connected') {
        setState(() { connected = true; connecting = false; });
      } else if (e == 'disconnected') {
        setState(() { connected = false; connecting = false; });
      } else if (e == 'connecting') {
        setState(() { connecting = true; });
      } else if (e.startsWith('log:')) {
        setState(() {
          logs.add(e.substring(4));
          if (logs.length > 500) logs.removeAt(0);
        });
      }
    });
    _load();
  }

  Future<void> _load() async {
    final p = await SharedPreferences.getInstance();
    final raw = p.getString('settings');
    if (raw != null) {
      try {
        s = Settings.fromMap(jsonDecode(raw));
      } catch (_) {}
    } else {
      // migrate old keys if present
      final oldDomains = p.getString('domains');
      if (oldDomains != null) s.domains = oldDomains.split(',').map((e) => e.trim()).where((e) => e.isNotEmpty).toList();
      s.key = p.getString('key') ?? '';
    }
    domainsCtrl.text = s.domains.join(', ');
    keyCtrl.text = s.key;
    listenIpCtrl.text = s.listenIp;
    listenPortCtrl.text = s.listenPort.toString();
    localDnsIpCtrl.text = s.localDnsIp;
    localDnsPortCtrl.text = s.localDnsPort.toString();
    resolversCtrl.text = s.resolvers.join(', ');
    dupCtrl.text = s.packetDuplication.toString();
    workersCtrl.text = s.rxTxWorkers.toString();
    setState(() => _loaded = true);
  }

  void _pullFromFields() {
    s.domains = domainsCtrl.text.split(',').map((e) => e.trim()).where((e) => e.isNotEmpty).toList();
    s.key = keyCtrl.text.trim();
    s.listenIp = listenIpCtrl.text.trim();
    s.listenPort = int.tryParse(listenPortCtrl.text.trim()) ?? s.listenPort;
    s.localDnsIp = localDnsIpCtrl.text.trim();
    s.localDnsPort = int.tryParse(localDnsPortCtrl.text.trim()) ?? s.localDnsPort;
    s.resolvers = resolversCtrl.text.split(',').map((e) => e.trim()).where((e) => e.isNotEmpty).toList();
    s.packetDuplication = int.tryParse(dupCtrl.text.trim()) ?? s.packetDuplication;
    s.rxTxWorkers = int.tryParse(workersCtrl.text.trim()) ?? s.rxTxWorkers;
  }

  Future<void> _save() async {
    _pullFromFields();
    final p = await SharedPreferences.getInstance();
    await p.setString('settings', jsonEncode(s.toMap()));
  }

  Future<void> _toggle() async {
    if (connected || connecting) {
      await TunnelService.stop();
      setState(() { connected = false; connecting = false; });
      return;
    }
    _pullFromFields();
    if (s.domains.isEmpty) {
      _snack('حداقل یک دامنهٔ تانل وارد کن');
      _tabs.animateTo(1);
      return;
    }
    if (s.key.isEmpty) {
      _snack('کلید رمزنگاری را وارد کن');
      _tabs.animateTo(1);
      return;
    }
    await _save();
    await TunnelService.requestPermission();
    await TunnelService.start(s);
    setState(() => connecting = true);
  }

  void _snack(String m) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ZarvandVPN'),
        bottom: TabBar(
          controller: _tabs,
          tabs: const [
            Tab(icon: Icon(Icons.shield_outlined), text: 'اتصال'),
            Tab(icon: Icon(Icons.tune), text: 'تنظیمات'),
            Tab(icon: Icon(Icons.terminal), text: 'لاگ'),
          ],
        ),
      ),
      body: !_loaded
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabs,
              children: [_connectTab(), _settingsTab(), _logTab()],
            ),
    );
  }

  Widget _connectTab() {
    final color = connected
        ? const Color(0xFF00B86B)
        : connecting
            ? Colors.orange
            : const Color(0xFFE5484D);
    final label = connected ? 'متصل' : connecting ? 'در حال اتصال…' : 'قطع';
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        children: [
          const SizedBox(height: 20),
          GestureDetector(
            onTap: _toggle,
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 250),
              width: 190,
              height: 190,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: RadialGradient(
                  colors: [color.withValues(alpha: 0.95), color.withValues(alpha: 0.55)],
                ),
                boxShadow: [
                  BoxShadow(color: color.withValues(alpha: 0.45), blurRadius: 40, spreadRadius: 4),
                ],
              ),
              child: Icon(
                connected
                    ? Icons.lock
                    : connecting
                        ? Icons.sync
                        : Icons.lock_open,
                size: 74,
                color: Colors.white,
              ),
            ),
          ),
          const SizedBox(height: 18),
          Text(label, style: const TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: Colors.white)),
          const SizedBox(height: 28),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _kv('دامنه‌ها', s.domains.isEmpty ? '—' : s.domains.join(' , ')),
                  _kv('سرورهای DNS', '${s.resolvers.length} مورد'),
                  _kv('روش رمزنگاری', _encName(s.encryptionMethod)),
                  _kv('حالت', s.protocol),
                  _kv('پروکسی محلی', '${s.listenIp}:${s.listenPort}'),
                  _kv('DNS محلی', s.localDns ? '${s.localDnsIp}:${s.localDnsPort}' : 'غیرفعال'),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _kv(String k, String v) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(
          children: [
            Expanded(child: Text(k, style: const TextStyle(color: Colors.white70))),
            Expanded(
              child: Text(v,
                  textAlign: TextAlign.left,
                  style: const TextStyle(fontWeight: FontWeight.w600)),
            ),
          ],
        ),
      );

  String _encName(int m) =>
      const ['بدون', 'XOR', 'ChaCha20', 'AES-128', 'AES-192', 'AES-256'][m.clamp(0, 5)];

  Widget _settingsTab() {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _section('تانل و امنیت'),
        _field(domainsCtrl, 'دامنه‌های تانل (با کاما جدا)'),
        _field(keyCtrl, 'کلید رمزنگاری (ENCRYPTION_KEY)', obscure: true),
        _dropdown('روش رمزنگاری', s.encryptionMethod,
            const {0: 'بدون', 1: 'XOR', 2: 'ChaCha20', 3: 'AES-128-GCM', 4: 'AES-192-GCM', 5: 'AES-256-GCM'},
            (v) => setState(() => s.encryptionMethod = v!)),
        _section('سرورها (Resolver DNS)'),
        _field(resolversCtrl, 'آی‌پی سرورهای DNS (با کاما جدا)'),
        _section('پروکسی محلی'),
        _dropdown('حالت', s.protocol == 'SOCKS5' ? 0 : 1,
            const {0: 'SOCKS5 (پیشنهادی)', 1: 'TCP'}, (v) => setState(() => s.protocol = v == 0 ? 'SOCKS5' : 'TCP')),
        Row(children: [
          Expanded(child: _field(listenIpCtrl, 'آی‌پی')),
          const SizedBox(width: 10),
          Expanded(child: _field(listenPortCtrl, 'پورت', number: true)),
        ]),
        _section('DNS محلی'),
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('فعال‌سازی DNS محلی'),
          value: s.localDns,
          onChanged: (v) => setState(() => s.localDns = v),
        ),
        if (s.localDns)
          Row(children: [
            Expanded(child: _field(localDnsIpCtrl, 'آی‌پی DNS')),
            const SizedBox(width: 10),
            Expanded(child: _field(localDnsPortCtrl, 'پورت DNS', number: true)),
          ]),
        _section('عملکرد'),
        _field(dupCtrl, 'تعداد تکرار بسته (1-6)', number: true),
        _field(workersCtrl, 'تعداد Worker ها (1-16)', number: true),
        _dropdown('فشرده‌سازی آپلود', s.uploadCompression,
            const {0: 'غیرفعال', 1: 'Gzip', 2: 'Zstd', 3: 'LZ4'}, (v) => setState(() => s.uploadCompression = v!)),
        _dropdown('فشرده‌سازی دانلود', s.downloadCompression,
            const {0: 'غیرفعال', 1: 'Gzip', 2: 'Zstd', 3: 'LZ4'}, (v) => setState(() => s.downloadCompression = v!)),
        _dropdown('سطح لاگ', const ['DEBUG', 'INFO', 'WARN', 'ERROR'].indexOf(s.logLevel),
            const {0: 'DEBUG', 1: 'INFO', 2: 'WARN', 3: 'ERROR'},
            (v) => setState(() => s.logLevel = const ['DEBUG', 'INFO', 'WARN', 'ERROR'][v!])),
        const SizedBox(height: 20),
        FilledButton.icon(
          onPressed: () async {
            await _save();
            _snack('تنظیمات ذخیره شد');
          },
          icon: const Icon(Icons.save),
          label: const Text('ذخیرهٔ تنظیمات'),
        ),
        const SizedBox(height: 30),
      ],
    );
  }

  Widget _section(String t) => Padding(
        padding: const EdgeInsets.only(top: 18, bottom: 6),
        child: Text(t,
            style: const TextStyle(
                fontSize: 15, fontWeight: FontWeight.bold, color: Color(0xFF7FA8FF))),
      );

  Widget _field(TextEditingController c, String label,
          {bool obscure = false, bool number = false}) =>
      Padding(
        padding: const EdgeInsets.symmetric(vertical: 5),
        child: TextField(
          controller: c,
          obscureText: obscure,
          keyboardType: number ? TextInputType.number : TextInputType.text,
          decoration: InputDecoration(labelText: label, border: const OutlineInputBorder()),
        ),
      );

  Widget _dropdown(String label, Object value, Map<int, String> items, ValueChanged<int?> onChanged) =>
      Padding(
        padding: const EdgeInsets.symmetric(vertical: 5),
        child: DropdownButtonFormField<int>(
          initialValue: items.keys.firstWhere((k) => items[k] == value || k == value, orElse: () => items.keys.first),
          decoration: InputDecoration(labelText: label, border: const OutlineInputBorder()),
          items: items.entries.map((e) => DropdownMenuItem(value: e.key, child: Text(e.value))).toList(),
          onChanged: onChanged,
        ),
      );

  Widget _logTab() {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(8),
          child: Row(
            children: [
              const Text('لاگ‌ها', style: TextStyle(fontWeight: FontWeight.bold)),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.delete_outline),
                onPressed: () => setState(() => logs.clear()),
              ),
            ],
          ),
        ),
        Expanded(
          child: Container(
            width: double.infinity,
            margin: const EdgeInsets.all(10),
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: Colors.black,
              borderRadius: BorderRadius.circular(10),
            ),
            child: logs.isEmpty
                ? const Center(
                    child: Text('لاگی نیست', style: TextStyle(color: Colors.white38)))
                : ListView.builder(
                    itemCount: logs.length,
                    itemBuilder: (_, i) => Text(logs[i],
                        style: const TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 11,
                            color: Color(0xFF4AE38B))),
                  ),
          ),
        ),
      ],
    );
  }
}
