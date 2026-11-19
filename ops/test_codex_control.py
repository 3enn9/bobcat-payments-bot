import importlib.util,pathlib,tempfile,unittest
spec=importlib.util.spec_from_file_location('control',pathlib.Path(__file__).with_name('codex-control.py'))
c=importlib.util.module_from_spec(spec);spec.loader.exec_module(c)
class ScopeTests(unittest.TestCase):
 def test_scope(self):
  for p in [('internal','invoice','pdf.go'),('go.mod',),('miniapp','src','App.tsx')]:self.assertTrue(c.allowed(p))
  for p in [('.env',),('data','private.go'),('ops','codex-control.py'),('miniapp','node_modules','x.js'),('.git','config')]:self.assertFalse(c.allowed(p))
 def test_snapshot_never_follows_file_symlink(self):
  with tempfile.TemporaryDirectory() as d:
   c.WORK=pathlib.Path(d)/'work';c.DATA=pathlib.Path(d)/'data';c.WORK.mkdir();c.DATA.mkdir();(c.WORK/'go.mod').symlink_to('/etc/passwd')
   with self.assertRaises(ValueError):c.snapshot()
 def test_snapshot_excludes_secrets(self):
  with tempfile.TemporaryDirectory() as d:
   c.WORK=pathlib.Path(d)/'work';c.DATA=pathlib.Path(d)/'data';c.WORK.mkdir();c.DATA.mkdir();(c.WORK/'go.mod').write_text('module test');(c.WORK/'.env').write_text('secret')
   stage,digest=c.snapshot();self.assertTrue((stage/'go.mod').exists());self.assertFalse((stage/'.env').exists());self.assertEqual(len(digest),64)
if __name__=='__main__':unittest.main()
