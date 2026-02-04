import random

import discord

from src.items.Item import Item, ItemRarity, ItemType


class IdentityScroll(Item):
    def __init__(self):
        super().__init__(
            name="Parchemin d'Identité",
            price=500,
            description="📜 Change ton surnom sur le serveur aléatoirement.",
            rarity=ItemRarity.epic,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        noms_rigolos = ["Guacamole Master", "Le Noob", "Pigeon", "Roi du Casino", "Banane Flambée"]
        new_nick = random.choice(noms_rigolos)
        try:
            await ctx.author.edit(nick=new_nick)
            await ctx.send(f"📜 Une magie opère... Tu t'appelles maintenant **{new_nick}** !")
            return True
        except discord.Forbidden:
            await ctx.send("❌ Le bot n'a pas la permission de changer ton pseudo (Admin ?).")
            return False
