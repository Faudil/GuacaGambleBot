import json
import os
from typing import Dict, Any

LOCALES_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), 'locales')
_translations: Dict[str, Dict[str, Any]] = {}

def load_locales():
    global _translations
    _translations.clear()
    if not os.path.exists(LOCALES_DIR):
        print(f"Directory {LOCALES_DIR} not found.")
        return
        
    for filename in os.listdir(LOCALES_DIR):
        if filename.endswith('.json'):
            lang_code = filename[:-5]
            filepath = os.path.join(LOCALES_DIR, filename)
            try:
                with open(filepath, 'r', encoding='utf-8') as f:
                    _translations[lang_code] = json.load(f)
            except Exception as e:
                print(f"Failed to load translations for {lang_code}: {e}")

def _get_nested(d: Dict[str, Any], keys: list) -> Any:
    for key in keys:
        if isinstance(d, dict) and key in d:
            d = d[key]
        else:
            return None
    return d

def t(key: str, lang: str = 'fr', **kwargs) -> str:
    if not _translations:
        load_locales()
        
    lang_dict = _translations.get(lang)
    if not lang_dict:
        lang_dict = _translations.get('fr', {})
        
    keys = key.split('.')
    val = _get_nested(lang_dict, keys)
    
    if val is None:
        if lang != 'fr':
            fr_dict = _translations.get('fr', {})
            val = _get_nested(fr_dict, keys)
            
        if val is None:
            return key
            
    if isinstance(val, str) and kwargs:
        try:
            return val.format(**kwargs)
        except KeyError as e:
            print(f"Missing formatting key {e} for translation string {key}")
            return val
    elif isinstance(val, list):
        return "\n".join(val) if all(isinstance(v, str) for v in val) else str(val)
        
    return str(val)

def get_pet_name(species: str, lang: str = 'fr') -> str:
    return t(f"pet_types.{species}.name", lang)

def get_item_name(internal_name: str, lang: str = 'fr') -> str:
    return t(f"items.{internal_name.lower()}.name", lang)

def get_item_description(internal_name: str, lang: str = 'fr') -> str:
    return t(f"items.{internal_name.lower()}.description", lang)

def get_rarity_name(rarity: str, lang: str = 'fr') -> str:
    return t(f"rarities.{rarity.lower()}", lang)

load_locales()
