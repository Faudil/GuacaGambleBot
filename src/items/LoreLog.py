import random

from src.database.balance import update_balance
from src.items.Item import ItemRarity, ItemType, Item
from src.utils.i18n import t
from src.database.settings import get_language


class DataDisk(Item):
    def __init__(self):
        super().__init__(
            name="Data Disk",
            price=50,
            description="Un disque mémoire Zénith corrompu. Contient d'anciens logs de la Strate.",
            rarity=ItemRarity.rare,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        messages_fr = [
            "Log 304 : La purification de l'Éther a échoué. Le Nexus est hors de contrôle.",
            "Directive 88 : L'évacuation des Strates inférieures est annulée. Scellez les portes.",
            "Journal de Kael : J'ai trouvé un générateur encore intact, mais les Mangeurs de Rouille rodent.",
            "Transmission perdue : '...n'essayez pas de descendre. Ils ont muté. Je répète, la Racine-Éther a...'",
            "Note de recherche : L'hybridation technobiologique est en cours. Le premier spécimen est stable, et affamé."
        ]
        messages_en = [
            "Log 304: Aether purification failed. The Nexus is out of control.",
            "Directive 88: Evacuation of the lower Strata is cancelled. Seal the blast doors.",
            "Kael's Journal: Found an intact generator, but the Rust-Eaters are prowling.",
            "Lost Transmission: '...do not attempt to descend. They have mutated. I repeat, the Aether-Root is...'",
            "Research Note: Technobiological hybridization is ongoing. The first specimen is stable, and hungry."
        ]
        
        from src.database.housing import update_inventory
        # Remove item from inventory
        success, _ = update_inventory(ctx.author.id, self.name, -1, limit=999)

        if not success:
             await ctx.send(f"❌ Erreur lors de l'utilisation de l'objet.")
             return False

        if lang == 'fr':
            msg = random.choice(messages_fr)
        else:
            msg = random.choice(messages_en)
            
        await ctx.send(f"💾 **Bip... Bzzzt... Lecture du Disque :**\n*\"{msg}\"*")
        return True


class OldJournal(Item):
    def __init__(self):
        super().__init__(
            name="Old Journal",
            price=30,
            description="Un carnet poussiéreux, écrit à la main par un survivant. Les pages sont couvertes de suie.",
            rarity=ItemRarity.common,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        messages_fr = [
            "Jour 42 : L'eau est rance aujourd'hui. J'entends encore les grondements du Nexus la nuit.",
            "Jour 50 : On a vu une bête près des Champs Communaux. Ce n'était pas naturel... mi-chair, mi-acier.",
            "La rouille s'infiltre partout. Même nos poumons sont lourds. Il faut prier le Zénith.",
            "Note sur l'Arche : Si quelqu'un trouve ça... le code du bunker secondaire est 0451.",
            "Je regrette l'époque d'avant La Mue. Avant que le ciel ne devienne vert d'Éther."
        ]
        messages_en = [
            "Day 42: The water is rancid today. I still hear the Nexus rumbling at night.",
            "Day 50: We saw a beast near the Communal Fields. It wasn't natural... half flesh, half steel.",
            "The rust gets everywhere. Even our lungs feel heavy. We must pray to the Zenith.",
            "Note on the Ark: If anyone finds this... the secondary bunker code is 0451.",
            "I miss the days before The Shedding. Before the sky turned Aether green."
        ]
        
        from src.database.housing import update_inventory
        # Remove item from inventory
        success, _ = update_inventory(ctx.author.id, self.name, -1, limit=999)

        if not success:
             await ctx.send(f"❌ Erreur lors de l'utilisation de l'objet.")
             return False

        if lang == 'fr':
            msg = random.choice(messages_fr)
        else:
            msg = random.choice(messages_en)

        await ctx.send(f"📖 **Tu feuillettes les pages fragiles :**\n*\"{msg}\"*")
        return True
