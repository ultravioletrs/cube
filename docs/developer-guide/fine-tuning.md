---
id: fine-tuning
title: Fine-Tuning Models
sidebar_position: 5
---

## Fine-Tuning LLMs in Cube AI

This guide explains how to fine-tune a base LLM (such as Qwen or Mistral) on a custom dataset and deploy the resulting model into Cube AI for inference.

The workflow follows the original Cube AI developer guide structure while providing concrete, reproducible examples.

---

## Fine-Tuning Workflow (Cube AI)

The fine-tuning process consists of the following high-level steps:

1. Install required dependencies  
2. Prepare and format the dataset  
3. (Optional) Apply LoRA adapters  
4. Train the model  
5. Export the model to GGUF format  
6. Upload the model to Cube AI  

Each step is described in detail below.

---

## Step 1: Install Dependencies

Ensure you are using **Python 3.8+**.

Required libraries:

- `transformers` – model and tokenizer handling  
- `datasets` – dataset loading and processing  
- `torch` – model training backend  
- `peft` – parameter-efficient fine-tuning (LoRA)  
- `trl` – supervised fine-tuning trainer  
- `unsloth` – optimized fine-tuning and LoRA integration  

Example installation (Google Colab):

```bash
pip install torch==2.5.0+cu121 -f https://download.pytorch.org/whl/cu121/torch_stable.html
pip install "unsloth[colab-new] @ git+https://github.com/unslothai/unsloth.git"
pip install --no-deps xformers "trl<0.9.0" peft accelerate bitsandbytes
```

If you are using Colab, you may also mount Google Drive to persist datasets and model outputs:

```python
from google.colab import drive
drive.mount('/content/drive')
```

---

## Step 2: Prepare and Format the Dataset

Fine-tuning requires data structured as **prompt–input–completion** pairs.

Example structure:

- **Prompt**: Instruction (e.g. *"Write a function to reverse a string"*)  
- **Input**: Optional context or constraints  
- **Completion**: Expected output (code or text)  

Example dataset loading and formatting:

```python
from datasets import load_dataset
from transformers import AutoTokenizer

dataset = load_dataset("csv", data_files="path/to/your/data.csv")
tokenizer = AutoTokenizer.from_pretrained("Qwen/Qwen2.5-Coder-1.5B")

template = """Below is an instruction, an input, and a response that completes the request.

### Instruction:
{}

### Input:
{}

### Response:
{}"""

def format_examples(examples):
    texts = [
        template.format(p, i, c) + tokenizer.eos_token
        for p, i, c in zip(examples["prompt"], examples["input"], examples["completion"])
    ]
    return {"text": texts}

formatted_dataset = dataset.map(format_examples, batched=True)
```

If your dataset comes from a Git repository, first consolidate the code into a single file (for example using **Repomix**) and then generate prompt–completion pairs.

---

## Step 3: Apply LoRA Adapters (Optional)

LoRA (Low-Rank Adaptation) enables parameter-efficient fine-tuning with reduced memory usage.

```python
from unsloth import FastLanguageModel

model = FastLanguageModel.get_peft_model(
    model,
    r=16,
    target_modules=[
        "q_proj", "k_proj", "v_proj",
        "o_proj", "gate_proj", "up_proj", "down_proj"
    ],
    lora_alpha=8,
    lora_dropout=0.1,
    bias="none",
    use_gradient_checkpointing="unsloth",
    random_state=3407,
)
```

---

## Step 4: Configure Training

Define training arguments based on available hardware and dataset size.

```python
from transformers import TrainingArguments

training_args = TrainingArguments(
    per_device_train_batch_size=1,
    gradient_accumulation_steps=8,
    num_train_epochs=2,
    learning_rate=5e-5,
    fp16=True,
    logging_steps=1,
    optim="adamw_8bit",
    output_dir="outputs",
)
```

---

## Step 5: Train the Model

For supervised fine-tuning, use `SFTTrainer`:

```python
from trl import SFTTrainer

trainer = SFTTrainer(
    model=model,
    tokenizer=tokenizer,
    train_dataset=formatted_dataset,
    dataset_text_field="text",
    max_seq_length=2048,
)

trainer.train()
```

Optional evaluation:

```python
trainer.evaluate()
```

---

## Step 6: Export the Model to GGUF

After training, export the model in **GGUF** format using `llama.cpp`.

```bash
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp
make
```

```python
model.push_to_hub_gguf(
    "hf_username/model_name",
    tokenizer=tokenizer,
    quantization_method=["q4_k_m", "q5_k_m", "q8_0", "f16"],
    token="hf_token",
)
```

---

## Step 7: Upload the Model to Cube AI

Once the model is exported in GGUF format, upload it to Cube AI for inference.

Refer to the **Private Model Upload** guide for detailed instructions:

👉 `docs/developer-guide/private-model-upload.md`

After uploading, the model becomes available for selection in the Cube AI Chat UI and inference APIs.

---

## Summary

This guide covered the complete Cube AI fine-tuning lifecycle:

- Dataset preparation  
- Optional LoRA-based fine-tuning  
- Training and evaluation  
- GGUF export  
- Deployment into Cube AI  

Following this workflow ensures your fine-tuned models integrate cleanly into Cube AI without fragmented or incomplete steps.
