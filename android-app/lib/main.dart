import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'tunnel_service.dart';

void main() => runApp(const ZarvandVPNApp());

class ZarvandVPNApp extends StatelessWidget {
  const ZarvandVPNApp({super.key});
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'ZarvandVPN',
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF2962FF)),
      ),
      darkTheme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF2962FF), brightness: Brightness.dark),
      ),
      home: const HomePage(),
    );
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  bool connected = false;
  final domainsCtrl = TextEditingController();
  final keyCtrl = TextEditingController();
  final serverCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    TunnelService.stateStream.listen((s) {
      if (!mounted) return;
      setState(() => connected = s == 'connected');
      if (s == 'log') TunnelService.drainLogs().forEach((l) => logs.add(l));
      if (mounted) setState(() {});
    });
    _load();
  }

  final List<String> logs = [];

  Future<void> _load() async {
    final p = await SharedPreferences.getInstance();
    domainsCtrl.text = p.getString('domains') ?? '';
    keyCtrl.text = p.getString('key') ?? '';
    serverCtrl.text = p.getString('server') ?? '';
  }

  Future<void> _save() async {
    final p = await SharedPreferences.getInstance();
    await p.setString('domains', domainsCtrl.text);
    await p.setString('key', keyCtrl.text);
    await p.setString('server', serverCtrl.text);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('ZarvandVPN')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Center(
              child: GestureDetector(
                onTap: () async {
                  await _save();
                  if (connected) {
                    await TunnelService.stop();
                  } else {
                    await TunnelService.start(
                      domains: domainsCtrl.text
                          .split(',')
                          .map((e) => e.trim())
                          .where((e) => e.isNotEmpty)
                          .toList(),
                      key: keyCtrl.text,
                      server: serverCtrl.text,
                    );
                  }
                  if (mounted) setState(() {});
                },
                child: Container(
                  width: 150,
                  height: 150,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: connected ? Colors.green.shade600 : Colors.red.shade400,
                    boxShadow: [
                      BoxShadow(
                        color: (connected ? Colors.green : Colors.red)
                            .withValues(alpha: 0.4),
                        blurRadius: 24,
                        spreadRadius: 2,
                      ),
                    ],
                  ),
                  child: Icon(
                    connected ? Icons.lock : Icons.lock_open,
                    size: 56,
                    color: Colors.white,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 10),
            Center(
              child: Text(connected ? 'متصل' : 'قطع',
                  style: Theme.of(context).textTheme.titleLarge),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: domainsCtrl,
              decoration: const InputDecoration(
                  labelText: 'دامنه‌های تانل (با کاما جدا کنید)'),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: keyCtrl,
              decoration: const InputDecoration(labelText: 'کلید رمزنگاری'),
              obscureText: true,
            ),
            const SizedBox(height: 8),
            TextField(
              controller: serverCtrl,
              decoration: const InputDecoration(
                  labelText: 'سرور (اختیاری - IP:Port)'),
            ),
            const SizedBox(height: 16),
            const Text('لاگ:', style: TextStyle(fontWeight: FontWeight.bold)),
            Expanded(
              child: Container(
                decoration: BoxDecoration(
                  color: Colors.black87,
                  borderRadius: BorderRadius.circular(8),
                ),
                padding: const EdgeInsets.all(8),
                child: ListView.builder(
                  itemCount: logs.length,
                  itemBuilder: (_, i) => Text(
                    logs[i],
                    style: const TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 11,
                        color: Colors.greenAccent),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
