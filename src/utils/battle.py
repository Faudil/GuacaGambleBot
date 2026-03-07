import asyncio
import discord
from src.models.Pet import Pet

async def simulate_battle(
    pet1: Pet,
    pet2: Pet,
    msg: discord.Message = None,
    embed: discord.Embed = None,
    update_embed_func = None,
    sleep_time: float = 0.5,
    send_messages: bool = True,
    log_size: int = 10,
    journal_title: str = "📜 **Journal de combat :**\n\n"
):
    """
    Simulate a battle between two pets.

    :param pet1: The first pet.
    :param pet2: The second pet.
    :param msg: The discord message to update (optional).
    :param embed: The discord embed to update (optional).
    :param update_embed_func: A function to call to update the embed fields (optional).
    :param sleep_time: Time to sleep between each attack animation.
    :param send_messages: Whether to send discord message updates.
    :param log_size: Maximum number of lines in the combat log.
    :param journal_title: Title of the combat log.
    :return: None
    """
    log = []
    turn = 1
    fighters = [pet1, pet2] if pet1.speed >= pet2.speed else [pet2, pet1]

    while pet1.is_alive and pet2.is_alive and turn <= 35:
        for i in range(2):
            attacker = fighters[i]
            defender = fighters[1 - i]

            if not attacker.is_alive:
                continue

            action_text = attacker.attack(defender)
            log.append(action_text)

            if len(log) > log_size:
                log.pop(0)

            if send_messages and msg and embed and update_embed_func:
                update_embed_func()
                embed.description = f"{journal_title}" + "\n".join(log)
                await msg.edit(embed=embed)
                await asyncio.sleep(sleep_time)

            if not defender.is_alive:
                break
        turn += 1
