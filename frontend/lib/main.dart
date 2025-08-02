import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:record/record.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:path_provider/path_provider.dart';
import 'package:http/http.dart' as http;

void main() {
  runApp(const SilenceApp());
}

class SilenceApp extends StatelessWidget {
  const SilenceApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Silence Audio Recorder',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
        useMaterial3: true,
      ),
      home: const AudioRecorderScreen(),
    );
  }
}

class AudioRecorderScreen extends StatefulWidget {
  const AudioRecorderScreen({super.key});

  @override
  State<AudioRecorderScreen> createState() => _AudioRecorderScreenState();
}

class _AudioRecorderScreenState extends State<AudioRecorderScreen> {
  late AudioRecorder _audioRecorder;
  bool _isRecording = false;
  bool _isProcessing = false;
  String? _recordedFilePath;
  List<String> _sseEvents = [];
  final String _backendUrl = 'http://localhost:8090';

  @override
  void initState() {
    super.initState();
    _audioRecorder = AudioRecorder();
  }

  @override
  void dispose() {
    _audioRecorder.dispose();
    super.dispose();
  }

  Future<void> _requestPermissions() async {
    await Permission.microphone.request();
  }

  Future<void> _startRecording() async {
    await _requestPermissions();

    if (await _audioRecorder.hasPermission()) {
      final directory = await getTemporaryDirectory();
      final filePath = '${directory.path}/audio_${DateTime.now().millisecondsSinceEpoch}.wav';

      await _audioRecorder.start(
        const RecordConfig(
          encoder: AudioEncoder.wav,
          sampleRate: 44100,
          bitRate: 128000,
        ),
        path: filePath,
      );

      setState(() {
        _isRecording = true;
        _recordedFilePath = filePath;
        _sseEvents.clear();
      });
    }
  }

  Future<void> _stopRecording() async {
    final path = await _audioRecorder.stop();
    setState(() {
      _isRecording = false;
      _recordedFilePath = path;
    });

    if (path != null) {
      await _sendAudioToBackend(path);
    }
  }

  Future<void> _sendAudioToBackend(String filePath) async {
    setState(() {
      _isProcessing = true;
      _sseEvents.add('Starting audio upload...');
    });

    try {
      final file = File(filePath);
      final request = http.MultipartRequest(
        'POST',
        Uri.parse('$_backendUrl/speak'),
      );

      request.files.add(
        await http.MultipartFile.fromPath('audio', filePath),
      );

      request.headers.addAll({
        'Accept': 'text/event-stream',
        'Cache-Control': 'no-cache',
      });

      final streamedResponse = await request.send();

      if (streamedResponse.statusCode == 200) {
        setState(() {
          _sseEvents.add('Connected to backend, receiving events...');
        });

        streamedResponse.stream
            .transform(utf8.decoder)
            .transform(const LineSplitter())
            .listen(
          (line) {
            if (line.startsWith('event:')) {
              final eventType = line.substring(6).trim();
              setState(() {
                _sseEvents.add('Event: $eventType');
              });
            } else if (line.startsWith('data:')) {
              final data = line.substring(5).trim();
              try {
                final jsonData = jsonDecode(data);
                setState(() {
                  _sseEvents.add('Data: ${jsonEncode(jsonData)}');
                });
              } catch (e) {
                setState(() {
                  _sseEvents.add('Raw data: $data');
                });
              }
            }
          },
          onDone: () {
            setState(() {
              _isProcessing = false;
              _sseEvents.add('Processing completed');
            });
          },
          onError: (error) {
            setState(() {
              _isProcessing = false;
              _sseEvents.add('Error: $error');
            });
          },
        );
      } else {
        setState(() {
          _isProcessing = false;
          _sseEvents.add('HTTP Error: ${streamedResponse.statusCode}');
        });
      }
    } catch (e) {
      setState(() {
        _isProcessing = false;
        _sseEvents.add('Exception: $e');
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Silence Audio Recorder'),
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16.0),
                child: Column(
                  children: [
                    Icon(
                      _isRecording ? Icons.mic : Icons.mic_none,
                      size: 64,
                      color: _isRecording ? Colors.red : Colors.grey,
                    ),
                    const SizedBox(height: 16),
                    Text(
                      _isRecording
                          ? 'Recording...'
                          : _isProcessing
                              ? 'Processing...'
                              : 'Ready to record',
                      style: Theme.of(context).textTheme.headlineSmall,
                    ),
                    const SizedBox(height: 16),
                    ElevatedButton(
                      onPressed: _isProcessing
                          ? null
                          : _isRecording
                              ? _stopRecording
                              : _startRecording,
                      child: Text(_isRecording ? 'Stop Recording' : 'Start Recording'),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            Expanded(
              child: Card(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Padding(
                      padding: const EdgeInsets.all(16.0),
                      child: Text(
                        'SSE Events Log',
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                    ),
                    const Divider(),
                    Expanded(
                      child: ListView.builder(
                        itemCount: _sseEvents.length,
                        itemBuilder: (context, index) {
                          return ListTile(
                            dense: true,
                            leading: Text(
                              '${index + 1}',
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                            title: Text(
                              _sseEvents[index],
                              style: const TextStyle(fontFamily: 'monospace'),
                            ),
                          );
                        },
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
