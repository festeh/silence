import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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
  String? _transcriptionResult;
  final String _backendUrl = const String.fromEnvironment('BACKEND_URL', defaultValue: 'http://localhost:8090');
  List<String> _exampleFiles = [];

  @override
  void initState() {
    super.initState();
    _audioRecorder = AudioRecorder();
    _loadExampleFiles();
  }

  Future<void> _loadExampleFiles() async {
    try {
      // Get the asset manifest to find all files in examples/
      final manifestContent = await rootBundle.loadString('AssetManifest.json');
      final Map<String, dynamic> manifestMap = json.decode(manifestContent);
      
      // Filter for .wav files in examples/ directory
      final exampleFiles = manifestMap.keys
          .where((String key) => key.startsWith('examples/') && key.endsWith('.wav'))
          .map((String path) => path.replaceFirst('examples/', ''))
          .toList();
      
      setState(() {
        _exampleFiles = exampleFiles;
      });
    } catch (e) {
      // If loading fails, keep the list empty and show loading state
      if (kDebugMode) {
        print('Error loading example files: $e');
      }
    }
  }

  @override
  void dispose() {
    _audioRecorder.dispose();
    super.dispose();
  }

  Future<void> _requestPermissions() async {
    if (!kIsWeb && Platform.isLinux) {
      return;
    }
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
        _transcriptionResult = null;
      });
    }
  }

  Future<void> _stopRecording() async {
    final path = await _audioRecorder.stop();
    setState(() {
      _isRecording = false;
    });

    if (path != null) {
      await _sendAudioToBackend(path);
    }
  }

  Future<void> _sendAudioToBackend(String filePath) async {
    setState(() {
      _isProcessing = true;
      _transcriptionResult = null;
    });

    try {
      final request = http.MultipartRequest(
        'POST',
        Uri.parse('$_backendUrl/speak'),
      );

      request.files.add(
        await http.MultipartFile.fromPath('audio', filePath),
      );

      final streamedResponse = await request.send();
      final responseBody = await streamedResponse.stream.bytesToString();

      if (streamedResponse.statusCode == 200) {
        final jsonResponse = jsonDecode(responseBody);
        setState(() {
          _isProcessing = false;
          _transcriptionResult = jsonResponse['transcribed_text'] ?? 'No transcription available';
        });
      } else {
        setState(() {
          _isProcessing = false;
          _transcriptionResult = 'Error: HTTP ${streamedResponse.statusCode}';
        });
      }
    } catch (e) {
      setState(() {
        _isProcessing = false;
        _transcriptionResult = 'Exception: $e';
      });
    }
  }

  Future<void> _sendExampleToBackend(String fileName) async {
    setState(() {
      _isProcessing = true;
      _transcriptionResult = null;
    });

    try {
      // Load the example file from assets
      final ByteData data = await rootBundle.load('examples/$fileName');
      final Uint8List bytes = data.buffer.asUint8List();

      final request = http.MultipartRequest(
        'POST',
        Uri.parse('$_backendUrl/speak'),
      );

      request.files.add(
        http.MultipartFile.fromBytes(
          'audio',
          bytes,
          filename: fileName,
        ),
      );

      final streamedResponse = await request.send();
      final responseBody = await streamedResponse.stream.bytesToString();

      if (streamedResponse.statusCode == 200) {
        final jsonResponse = jsonDecode(responseBody);
        setState(() {
          _isProcessing = false;
          _transcriptionResult = jsonResponse['text'] ?? 'No transcription available';
        });
      } else {
        setState(() {
          _isProcessing = false;
          _transcriptionResult = 'Error: HTTP ${streamedResponse.statusCode}';
        });
      }
    } catch (e) {
      setState(() {
        _isProcessing = false;
        _transcriptionResult = 'Exception: $e';
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
            Card(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: Text(
                      'Example Audio Files',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                  ),
                  const Divider(),
                  if (_exampleFiles.isEmpty)
                    const Padding(
                      padding: EdgeInsets.all(16.0),
                      child: Center(
                        child: Text('Loading example files...'),
                      ),
                    )
                  else
                    ..._exampleFiles.map((fileName) => ListTile(
                      leading: const Icon(Icons.audiotrack),
                      title: Text(fileName),
                      trailing: _isProcessing 
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.play_arrow),
                      onTap: _isProcessing ? null : () => _sendExampleToBackend(fileName),
                    )),
                ],
              ),
            ),
            const SizedBox(height: 16),
            if (_transcriptionResult != null)
              Expanded(
                child: Card(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Padding(
                        padding: const EdgeInsets.all(16.0),
                        child: Text(
                          'Transcription Result',
                          style: Theme.of(context).textTheme.titleLarge,
                        ),
                      ),
                      const Divider(),
                      Expanded(
                        child: Padding(
                          padding: const EdgeInsets.all(16.0),
                          child: SelectableText(
                            _transcriptionResult!,
                            style: Theme.of(context).textTheme.bodyLarge,
                          ),
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
