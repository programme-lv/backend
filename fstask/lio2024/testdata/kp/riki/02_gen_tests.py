import os
import subprocess
import shutil
import tempfile
import csv

script_dir = os.path.dirname(os.path.abspath(__file__))
gen_src_path = os.path.join(script_dir, "generator.cpp")
sol_src_path = os.path.join(script_dir, "../risin/kpetrucena-ac.cpp")

if not os.path.isfile(gen_src_path):
    print("gen cpp not found")
    exit(1)

if not os.path.isfile(sol_src_path):
    print("sol cpp not found")
    exit(1)

temp_dir = tempfile.mkdtemp()
gen_exe_path = os.path.join(temp_dir, "gen")
sol_exe_path = os.path.join(temp_dir, "sol")

test_dir = os.path.join(script_dir, "../testi")

print("Compiling generator and solution...")
subprocess.run(["g++", "-std=c++17", "-Wall", "-Wextra", "-Wpedantic", "-Werror", "-O2", "-o", gen_exe_path, gen_src_path], check=True)
subprocess.run(["g++", "-std=c++17", "-Wall", "-Wextra", "-Wpedantic", "-Werror", "-O2", "-o", sol_exe_path, sol_src_path], check=True)
print("Compiled successfully")

os.chdir(test_dir)
shutil.rmtree(test_dir)
os.makedirs(test_dir)

with open(os.path.join(script_dir, "01_param_list.csv"), "r") as f:
    reader = csv.reader(f, delimiter="\t")
    next(reader)  # skip header
    for row in reader:
        N, M, K, OK, T, group, subtask = row
        
        print(f"Generating test {group}...")
        
        in_name = f"kp.i{group}"
        out_name = f"kp.o{group}"
        in_path = os.path.join(test_dir, in_name)
        out_path = os.path.join(test_dir, out_name)

        print(f"{group} -> {in_name}")
    
        with open(in_path, 'w') as outfile:
            subprocess.run([gen_exe_path, N, M, K, OK, T], stdout=outfile)

# copy og test inputs
og_tests_dir = os.path.join(script_dir, "og_tests")
for test in os.listdir(og_tests_dir):
    in_name = test
    if ".o" in test:
        continue
    shutil.copyfile(os.path.join(og_tests_dir, in_name), os.path.join(test_dir, in_name))

# generate outputs from inputs
for test in os.listdir(test_dir):
    in_name = test
    out_name = test.replace(".i", ".o")
    in_path = os.path.join(test_dir, in_name)
    out_path = os.path.join(test_dir, out_name)
    
    print(f"{in_name} -> {out_name}")

    with open(in_path, 'r') as infile, open(out_path, 'w') as outfile:
        subprocess.run([sol_exe_path], stdin=infile, stdout=outfile)

# copy og test outputs
for test in os.listdir(og_tests_dir):
    out_name = test
    if ".i" in test:
        continue
    if ".o" not in test:
        continue
    shutil.copyfile(os.path.join(og_tests_dir, out_name), os.path.join(test_dir, out_name))

shutil.rmtree(temp_dir)
